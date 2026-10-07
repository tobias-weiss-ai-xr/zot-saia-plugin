package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/patriceckhart/zot/packages/agent/ext"

	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/models"
)

// zotHome mirrors zot's own data-dir resolution (packages/agent/config.go,
// ZotHome) so this extension targets the exact same auth.json the agent uses.
func zotHome() string {
	if v := os.Getenv("ZOT_HOME"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "zot")
	}
	switch runtime.GOOS {
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "zot")
		}
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "zot")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "state", "zot")
	}
	return ".zot"
}

func authPath() string       { return filepath.Join(zotHome(), "auth.json") }
func userModelsPath() string { return filepath.Join(zotHome(), "models.json") }

// maskKey shortens an API key for display so it is never echoed in full.
func maskKey(k string) string {
	if k == "" {
		return "none"
	}
	if len(k) <= 8 {
		return "••••"
	}
	return k[:4] + "…" + k[len(k)-4:]
}

// saiaCreds is the shape zot persists for a custom provider under
// "additional_api_key_creds". We also write an extra _demo marker only in
// memory for masked display; the on-disk object keeps only zot's fields.
type saiaCreds struct {
	APIKey        string `json:"api_key,omitempty"`
	APIKeyCommand any    `json:"api_key_command,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	OAuth         any    `json:"oauth,omitempty"`
}

// readCreds loads zot's auth.json as a generic JSON object so we can rewrite
// only the "additional_api_key_creds.saia" slot and preserve every other
// provider and every other field verbatim (anthropic, openai, oauth, …).
func readCreds(path string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	root := map[string]json.RawMessage{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &root); err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	additional := map[string]json.RawMessage{}
	if raw, ok := root["additional_api_key_creds"]; ok {
		if err := json.Unmarshal(raw, &additional); err != nil {
			return nil, nil, fmt.Errorf("parse additional_api_key_creds in %s: %w", path, err)
		}
	}
	return root, additional, nil
}

// storeSAIAKey writes (merging) the saia credential into zot's auth.json with
// mode 0600, matching what the agent would persist via SetEndpointCredential.
func storeSAIAKey(key string) error {
	path := authPath()
	root, additional, err := readCreds(path)
	if err != nil {
		return err
	}

	// The stored object only carries zot's own fields; the added "_masked" key
	// is for our display response and is dropped here.
	creds, _ := json.Marshal(struct {
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
	}{APIKey: key, BaseURL: models.BaseURL})
	additional[models.ProviderPrefix] = creds
	root["additional_api_key_creds"], _ = json.Marshal(additional)

	if b, err := json.MarshalIndent(root, "", "  "); err != nil {
		return err
	} else {
		b = append(b, '\n')
		if err := os.MkdirAll(zotHome(), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o600)
	}
}

// loginSAIA stores the SAIA API key (and canonical base URL) in zot's
// auth.json under the custom-provider credential slot for "saia".
func loginSAIA(args string) ext.Response {
	key := strings.TrimSpace(args)
	if key == "" {
		key = os.Getenv(models.APIKeyEnv)
	}
	if key == "" {
		return ext.Errorf("no API key supplied — pass /saia-login <key> or export %s and run /saia-login", models.APIKeyEnv)
	}

	if err := storeSAIAKey(key); err != nil {
		return ext.Errorf("failed to write %s: %v", authPath(), err)
	}

	lines := []string{
		fmt.Sprintf("SAIA auth configured for %s", models.ProviderPrefix),
		fmt.Sprintf("  base URL : %s", models.BaseURL),
		fmt.Sprintf("  stored at: %s", authPath()),
		fmt.Sprintf("  api key  : %s (stored; not shown in full)", maskKey(key)),
	}
	if env := os.Getenv(models.APIKeyEnv); env != "" {
		if env != key {
			lines = append(lines,
				fmt.Sprintf("  %s env is set (%s) and DIFFERS from the stored key — the env", models.APIKeyEnv, maskKey(env)),
				"  value takes precedence at runtime. Unset it (`unset "+models.APIKeyEnv+"`) or",
				"  re-export it to match, or saia will 401.",
			)
		} else {
			lines = append(lines, fmt.Sprintf("  %s env matches the stored key — good.", models.APIKeyEnv))
		}
	} else {
		lines = append(lines, fmt.Sprintf("  %s env not set — the stored key will be used.", models.APIKeyEnv))
	}
	return ext.Prompt(strings.Join(lines, "\n"))
}

// authStatus reports how saia authentication will resolve for the active zot
// install: the data dir, what (if anything) is stored in auth.json, whether a
// SAIA_API_KEY env var is present and conflicts, and whether the saia provider
// is registered in $ZOT_HOME/models.json.
func authStatus(_ string) ext.Response {
	lines := []string{"SAIA authentication status", "-------------------------"}

	home := zotHome()
	lines = append(lines, fmt.Sprintf("  zot data dir     : %s", home))

	// Resolve the stored saia credential by merging raw JSON.
	root := map[string]json.RawMessage{}
	key, baseURL := "", ""
	if b, err := os.ReadFile(authPath()); err == nil {
		_ = json.Unmarshal(b, &root)
		var add map[string]json.RawMessage
		if raw, ok := root["additional_api_key_creds"]; ok && json.Unmarshal(raw, &add) == nil {
			if c, ok := add[models.ProviderPrefix]; ok {
				var sc struct {
					APIKey  string `json:"api_key"`
					BaseURL string `json:"base_url"`
				}
				_ = json.Unmarshal(c, &sc)
				key, baseURL = sc.APIKey, sc.BaseURL
			}
		}
	}
	lines = append(lines,
		fmt.Sprintf("  auth.json key    : %s", maskKey(key)),
		fmt.Sprintf("  auth.json base   : %s", nonEmpty(baseURL, "none")),
	)

	env := os.Getenv(models.APIKeyEnv)
	method := "auth.json"
	if env != "" {
		method = fmt.Sprintf("%s env", models.APIKeyEnv)
		if key != "" && env != key {
			lines = append(lines, fmt.Sprintf("  !! %s env (%s) differs from auth.json (%s) — env wins at runtime", models.APIKeyEnv, maskKey(env), maskKey(key)))
		}
	}
	lines = append(lines,
		fmt.Sprintf("  env %-12s : %s", models.APIKeyEnv, maskKey(env)),
		fmt.Sprintf("  would use       : %s", method),
	)

	// Is the saia provider registered in $ZOT_HOME/models.json?
	reg := "NOT registered"
	if b, err := os.ReadFile(userModelsPath()); err == nil && strings.Contains(string(b), `"saia"`) &&
		strings.Contains(string(b), `"providers"`) {
		reg = "registered"
	}
	lines = append(lines,
		fmt.Sprintf("  models.json      : %s (%s)", reg, userModelsPath()),
		"",
		"  If models.json shows NOT registered, copy this plugin's models.json",
		"  into $ZOT_HOME/models.json so the saia provider is available.",
	)

	return ext.Prompt(strings.Join(lines, "\n"))
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// installModelsJSON writes (merging) the saia provider registration into
// $ZOT_HOME/models.json using the plugin's embedded catalog, so the user never
// has to hand-copy a models.json to query saia. Existing entries in the
// top-level "providers" map are preserved, and any existing saia entry is
// replaced with the current catalog (deepseek-v4-flash-0731 first).
func installModelsJSON() error {
	path := userModelsPath()

	root := map[string]json.RawMessage{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &root)
	}

	providers := map[string]json.RawMessage{}
	if raw, ok := root["providers"]; ok {
		_ = json.Unmarshal(raw, &providers)
	}

	saia := struct {
		BaseURL string            `json:"baseUrl"`
		API     string            `json:"api"`
		Models  []models.OutModel `json:"models"`
	}{BaseURL: models.BaseURL, API: "openai", Models: models.OutModels()}
	providers[models.ProviderPrefix], _ = json.Marshal(saia)
	root["providers"], _ = json.Marshal(providers)

	if b, err := json.MarshalIndent(root, "", "  "); err != nil {
		return err
	} else {
		b = append(b, '\n')
		if err := os.MkdirAll(zotHome(), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o644)
	}
}

// installModelsSAIA is the /saia-install-models command handler.
func installModelsSAIA(_ string) ext.Response {
	if err := installModelsJSON(); err != nil {
		return ext.Errorf("failed to write %s: %v", userModelsPath(), err)
	}
	return ext.Prompt(fmt.Sprintf(
		"SAIA provider registered\n  wrote: %s\n  models: %d (deepseek-v4-flash-0731 first = default)\n  base : %s\n",
		userModelsPath(), len(models.All), models.BaseURL))
}
