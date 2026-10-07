package models_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/internal/wiretest"
)

// resolveAuthBin resolves the extension binary via the shared binaryPath flag.
func resolveAuthBin(t *testing.T) string {
	t.Helper()
	bin := *binaryPath
	if bin == "" {
		for _, c := range []string{"../zot-saia-plugin", "../../zot-saia-plugin"} {
			if info, err := os.Stat(c); err == nil && info.Mode()&0o111 != 0 {
				bin = c
				break
			}
		}
	}
	if bin == "" {
		t.Skip("skipping auth addon wire tests: build the binary first (./build.sh)")
	}
	return bin
}

func authRegistrations(t *testing.T, sess *wiretest.Session) (login, status bool) {
	for _, f := range sess.Out {
		if f["type"] == "register_command" {
			switch n, _ := f["name"].(string); n {
			case "saia-login":
				login = true
			case "saia-authstatus":
				status = true
			}
		}
	}
	return
}

// TestAuthAddonLoginWire drives the real extension over the wire protocol:
// /saia-login must write zot's auth.json in the process's ZOT_HOME and report
// success without echoing the full key.
func TestAuthAddonLoginWire(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZOT_HOME", home)
	t.Setenv("XDG_STATE_HOME", "/should/not/win")
	os.Unsetenv("SAIA_API_KEY")

	sess := wiretest.NewSession(t, resolveAuthBin(t))
	defer sess.Shutdown()

	login, status := authRegistrations(t, sess)
	if !login {
		t.Error("/saia-login command not registered")
	}
	if !status {
		t.Error("/saia-authstatus command not registered")
	}

	key := "a163c2d61e20e82c763b3e9e9b63a0d9"
	resp := sess.InvokeCommand("saia-login", key)
	if action, _ := resp["action"].(string); action != "prompt" {
		t.Fatalf("login response action = %q, want prompt", action)
	}
	prompt, _ := resp["prompt"].(string)
	if !strings.Contains(prompt, "configured") {
		t.Errorf("login prompt missing 'configured': %q", prompt)
	}
	if strings.Contains(prompt, key) {
		t.Error("login prompt leaked the full API key")
	}

	// auth.json written with the saia credential + canonical base URL in the
	// right (additional_api_key_creds) slot, replacing nothing else.
	b, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("auth.json not written: %v", err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("auth.json no longer parses: %v", err)
	}
	var add map[string]json.RawMessage
	if err := json.Unmarshal(root["additional_api_key_creds"], &add); err != nil {
		t.Fatalf("additional_api_key_creds missing: %v", err)
	}
	var saia struct {
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
	}
	if err := json.Unmarshal(add["saia"], &saia); err != nil {
		t.Fatalf("saia cred missing: %v", err)
	}
	if saia.APIKey != key {
		t.Errorf("stored api_key = %q", saia.APIKey)
	}
	if saia.BaseURL != "https://chat-ai.academiccloud.de/v1" {
		t.Errorf("stored base_url = %q", saia.BaseURL)
	}

	// authstatus reports the stored key + the auth.json method.
	st := sess.InvokeCommand("saia-authstatus", "")
	prompt, _ = st["prompt"].(string)
	if !strings.Contains(prompt, "auth.json key") || !strings.Contains(prompt, "would use") {
		t.Errorf("authstatus missing expected sections: %q", prompt)
	}
	if strings.Contains(prompt, key) {
		t.Error("authstatus leaked the full API key")
	}
}

// TestAuthAddonEnvConflictWire proves the addon surfaces zot's
// env-overrides-auth.json precedence so a stray SAIA_API_KEY cannot silently
// win over a stored key.
func TestAuthAddonEnvConflictWire(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZOT_HOME", home)
	t.Setenv("SAIA_API_KEY", "c695b49145b153a3f85e72fcef430b9d") // a different (rotated) key

	sess := wiretest.NewSession(t, resolveAuthBin(t))
	defer sess.Shutdown()

	// Store a good key via /saia-login (the env is set to a different, rotated
	// key — this is the exact footgun zot has). Then status must warn that the
	// env wins over the stored key.
	stored := "a163c2d61e20e82c763b3e9e9b63a0d9"
	if resp := sess.InvokeCommand("saia-login", stored); resp["action"] != "prompt" {
		t.Fatalf("login failed: %v", resp)
	}

	resp := sess.InvokeCommand("saia-authstatus", "")
	prompt, _ := resp["prompt"].(string)
	if !strings.Contains(prompt, "differs from auth.json") && !strings.Contains(prompt, "env wins") {
		t.Errorf("authstatus should flag the env/auth.json conflict: %q", prompt)
	}
	if !strings.Contains(prompt, "env") {
		t.Errorf("authstatus missing env mention: %q", prompt)
	}
}

// TestAuthAddonInstallModelsWire drives /saia-install-models over the wire
// protocol and asserts it writes a correct, usable providers.saia block into
// $ZOT_HOME/models.json (14 models, deepseek-v4-flash-0731 first = default).
func TestAuthAddonInstallModelsWire(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZOT_HOME", home)
	t.Setenv("XDG_STATE_HOME", "/should/not/win")
	os.Unsetenv("SAIA_API_KEY")

	sess := wiretest.NewSession(t, resolveAuthBin(t))
	defer sess.Shutdown()

	resp := sess.InvokeCommand("saia-install-models", "")
	if action, _ := resp["action"].(string); action != "prompt" {
		t.Fatalf("install-models response action = %q, want prompt", action)
	}
	prompt, _ := resp["prompt"].(string)
	if !strings.Contains(prompt, "registered") {
		t.Errorf("install-models prompt missing 'registered': %q", prompt)
	}

	b, err := os.ReadFile(filepath.Join(home, "models.json"))
	if err != nil {
		t.Fatalf("models.json not written: %v", err)
	}
	var root struct {
		Providers map[string]struct {
			BaseURL string `json:"baseUrl"`
			API     string `json:"api"`
			Models  []struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ContextWindow int    `json:"contextWindow"`
				MaxTokens     int    `json:"maxTokens"`
				Reasoning     bool   `json:"reasoning"`
			} `json:"models"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("models.json no longer parses: %v", err)
	}
	saia, ok := root.Providers["saia"]
	if !ok {
		t.Fatalf("providers.saia missing: %s", b)
	}
	if saia.BaseURL != "https://chat-ai.academiccloud.de/v1" || saia.API != "openai" {
		t.Errorf("providers.saia baseUrl/api wrong: %s", b)
	}
	if len(saia.Models) != 14 {
		t.Errorf("model count = %d, want 14", len(saia.Models))
	}
	first := saia.Models[0]
	if first.ID != "deepseek-v4-flash-0731" {
		t.Errorf("first model = %q, want deepseek-v4-flash-0731 (default)", first.ID)
	}
	if first.ContextWindow != 1000000 || first.MaxTokens != 32768 {
		t.Errorf("deepseek contextWindow/maxTokens = %d/%d, want 1000000/32768", first.ContextWindow, first.MaxTokens)
	}
}
