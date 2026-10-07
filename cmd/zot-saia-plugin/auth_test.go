package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/models"
)

func TestMaskKey(t *testing.T) {
	if got := maskKey(""); got != "none" {
		t.Fatalf("maskKey(\"\") = %q, want none", got)
	}
	if got := maskKey("abc"); got != "••••" {
		t.Fatalf("maskKey(short) = %q, want ••••", got)
	}
	got := maskKey("a163c2d61e20e82c763b3e9e9b63a0d9")
	if strings.Contains(got, "a163c2d61e20e82c763b3e9e9b63a0d9") {
		t.Fatalf("maskKey leaked the full key: %q", got)
	}
}

func TestZotHomePrecedence(t *testing.T) {
	t.Setenv("ZOT_HOME", "/tmp/zot-unit")
	t.Setenv("XDG_STATE_HOME", "/should/not/win")
	if got := zotHome(); got != "/tmp/zot-unit" {
		t.Fatalf("ZOT_HOME override = %q", got)
	}
}

func TestStoreSAIAKeyMergesAndPreserves(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZOT_HOME", home)
	t.Setenv("XDG_STATE_HOME", "/should/not/win")

	// Pre-seed an auth.json with an unrelated built-in provider and a second
	// custom provider (with an oauth-ish extra field) to prove we merge rather
	// than clobber.
	seed := `{
  "anthropic": {"api_key": "sk-ant-existing"},
  "additional_api_key_creds": {
    "other": {"api_key": "sk-other", "base_url": "https://x.example/v1", "oauth": {"token": "t"}}
  }
}`
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	key := "a163c2d61e20e82c763b3e9e9b63a0d9"
	if err := storeSAIAKey(key); err != nil {
		t.Fatalf("storeSAIAKey: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("written auth.json no longer parses: %v", err)
	}

	// Other providers preserved.
	var an struct {
		APIKey string `json:"api_key"`
	}
	json.Unmarshal(root["anthropic"], &an)
	if an.APIKey != "sk-ant-existing" {
		t.Errorf("anthropic not preserved: %s", b)
	}
	var add map[string]json.RawMessage
	json.Unmarshal(root["additional_api_key_creds"], &add)
	var other struct {
		APIKey string `json:"api_key"`
		OAuth  struct {
			Token string `json:"token"`
		} `json:"oauth"`
	}
	json.Unmarshal(add["other"], &other)
	if other.APIKey != "sk-other" || other.OAuth.Token != "t" {
		t.Errorf("other custom provider not preserved: %s", add["other"])
	}

	// saia slot written correctly.
	var saia struct {
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
	}
	json.Unmarshal(add["saia"], &saia)
	if saia.APIKey != key || saia.BaseURL != models.BaseURL {
		t.Errorf("saia cred wrong: api_key=%q base_url=%q", saia.APIKey, saia.BaseURL)
	}

	// File is 0600 and the full key is present on disk (so it works), but the
	// display helper masks it.
	fi, err := os.Stat(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("auth.json mode = %v, want 0600", fi.Mode().Perm())
	}
	if !strings.Contains(string(b), key) {
		t.Errorf("stored key not on disk")
	}
}
