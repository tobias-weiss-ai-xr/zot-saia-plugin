package models_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/internal/wiretest"
	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/models"
)

// ---------------------------------------------------------------------------
// Test flags
// ---------------------------------------------------------------------------

var (
	binaryPath = flag.String("binary", "", "path to zot-saia-plugin binary for integration tests")
)

// ---------------------------------------------------------------------------
// Model catalog: table-driven tests
// ---------------------------------------------------------------------------

func TestAllModelsCount(t *testing.T) {
	if got, want := len(models.All), 6; got != want {
		t.Fatalf("expected %d models, got %d", want, got)
	}
}

func TestModelIDsAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(models.All))
	for _, m := range models.All {
		if seen[m.ID] {
			t.Errorf("duplicate model ID: %q", m.ID)
		}
		seen[m.ID] = true
	}
}

func TestModelNamesAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(models.All))
	for _, m := range models.All {
		if seen[m.Name] {
			t.Errorf("duplicate model name: %q", m.Name)
		}
		seen[m.Name] = true
	}
}

func TestModelProperties(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantCtx  string
		wantName string
	}{
		{"glm", "glm-4.7", "128K", "GLM 4.7"},
		{"qwen397", "qwen3.5-397b-a17b", "128K", "Qwen 3.5 397B"},
		{"qwen122", "qwen3.5-122b-a10b", "128K", "Qwen 3.5 122B"},
		{"devstral", "devstral-2-123b-instruct-2512", "128K", "DevStral 2 123B"},
		{"gptoss", "openai-gpt-oss-120b", "128K", "GPT-OSS 120B"},
		{"qwen36", "qwen3.6-27b", "128K", "Qwen 3.6 27B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := models.FindByID(tt.id)
			if m == nil {
				t.Fatalf("FindByID(%q) = nil", tt.id)
			}
			if m.Ctx != tt.wantCtx {
				t.Errorf("Ctx = %q, want %q", m.Ctx, tt.wantCtx)
			}
			if m.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", m.Name, tt.wantName)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FindByID / FindByName: edge cases
// ---------------------------------------------------------------------------

func TestFindByID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string // empty = nil
	}{
		{"found", "glm-4.7", "GLM 4.7"},
		{"not found", "nonexistent", ""},
		{"empty string", "", ""},
		{"partial match", "glm", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := models.FindByID(tt.id)
			if tt.want == "" {
				if m != nil {
					t.Fatalf("expected nil, got %+v", m)
				}
				return
			}
			if m == nil {
				t.Fatal("expected non-nil")
			}
			if m.Name != tt.want {
				t.Errorf("Name = %q", m.Name)
			}
		})
	}
}

func TestFindByName(t *testing.T) {
	tests := []struct {
		name     string
		search    string
		wantFound bool
		wantID    string
	}{
		{"found", "GLM 4.7", true, "glm-4.7"},
		{"not found", "nonexistent", false, ""},
		{"empty", "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := models.FindByName(tt.search)
			if !tt.wantFound {
				if m != nil {
					t.Fatalf("expected nil, got %+v", m)
				}
				return
			}
			if m == nil {
				t.Fatal("expected non-nil")
			}
			if m.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", m.ID, tt.wantID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FullID
// ---------------------------------------------------------------------------

func TestModelFullID(t *testing.T) {
	for _, m := range models.All {
		got := m.FullID()
		if !strings.HasPrefix(got, models.ProviderPrefix+"/") {
			t.Errorf("FullID(%q) = %q, missing prefix", m.ID, got)
		}
		if !strings.HasSuffix(got, m.ID) {
			t.Errorf("FullID(%q) = %q, missing model ID", m.ID, got)
		}
	}
}

// ---------------------------------------------------------------------------
// BuildPrompt: table-driven
// ---------------------------------------------------------------------------

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name   string
		models []models.Model
	}{
		{"all models", models.All},
		{"empty slice", nil},
		{"single model", []models.Model{{ID: "x", Name: "X", Ctx: "64K"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.BuildPrompt(tt.models)
			if !strings.Contains(got, models.ProviderName) {
				t.Error("missing provider name")
			}
			if !strings.Contains(got, "### Usage") {
				t.Error("missing usage section")
			}
			if !strings.Contains(got, "SAIA_API_KEY") {
				t.Error("missing API key env var")
			}
			for _, m := range tt.models {
				if !strings.Contains(got, m.FullID()) {
					t.Errorf("missing model %q", m.FullID())
				}
			}
		})
	}
}

func TestPromptEqualsBuildPromptAll(t *testing.T) {
	if got, want := models.Prompt(), models.BuildPrompt(models.All); got != want {
		t.Error("Prompt() != BuildPrompt(All)")
	}
}

// ---------------------------------------------------------------------------
// Golden file test
// ---------------------------------------------------------------------------

func TestPromptGolden(t *testing.T) {
	wiretest.Golden(t, "testdata/prompt.golden", models.Prompt())
}

// ---------------------------------------------------------------------------
// models.json round-trip
// ---------------------------------------------------------------------------

func TestModelsJSONRoundTrip(t *testing.T) {
	b, err := os.ReadFile("../models.json")
	if err != nil {
		t.Skip("models.json not found")
	}

	var cfg struct {
		AdditionalProviders map[string]struct {
			Models []struct {
				ID string `json:"id"`
			} `json:"models"`
		} `json:"additional_providers"`
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("parse models.json: %v", err)
	}

	saia, ok := cfg.AdditionalProviders["saia"]
	if !ok {
		t.Fatal("models.json missing saia provider")
	}

	if got, want := len(saia.Models), len(models.All); got != want {
		t.Fatalf("models.json has %d models, code has %d", got, want)
	}

	for i, jm := range saia.Models {
		if jm.ID != models.All[i].ID {
			t.Errorf("model[%d] ID: json=%q, code=%q", i, jm.ID, models.All[i].ID)
		}
	}
}

// ---------------------------------------------------------------------------
// Virtual FS test
// ---------------------------------------------------------------------------

func TestModelsViaVirtualFS(t *testing.T) {
	content, err := json.Marshal(map[string]any{
		"additional_providers": map[string]any{
			"saia": map[string]any{
				"api":      "openai-completions",
				"base_url": models.BaseURL,
				"api_key_env": models.APIKeyEnv,
				"models": []map[string]any{
					{"id": "glm-4.7", "name": "GLM 4.7", "context_window": 131072, "reasoning": true},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	fs := fstest.MapFS{
		"models.json": &fstest.MapFile{Data: content},
	}

	b, err := fs.ReadFile("models.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("empty read")
	}
}

// ---------------------------------------------------------------------------
// Fuzz: BuildPrompt
// ---------------------------------------------------------------------------

func FuzzBuildPrompt(f *testing.F) {
	f.Add([]byte(`[{"id":"a","name":"A","ctx":"1K"}]`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var ms []models.Model
		if err := json.Unmarshal(data, &ms); err != nil {
			t.Skip()
		}
		out := models.BuildPrompt(ms)
		if len(ms) > 0 && !strings.Contains(out, "## ") {
			t.Error("heading missing")
		}
		if strings.Contains(out, "\r") {
			t.Error("carriage return in prompt")
		}
	})
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	tests := []struct {
		name  string
		got   string
		check func(string) bool
	}{
		{"ProviderPrefix", models.ProviderPrefix, func(s string) bool { return s == "saia" }},
		{"ProviderName", models.ProviderName, func(s string) bool { return strings.Contains(s, "SAIA") }},
		{"BaseURL", models.BaseURL, func(s string) bool { return strings.HasPrefix(s, "https://") }},
		{"APIKeyEnv", models.APIKeyEnv, func(s string) bool { return strings.HasSuffix(s, "API_KEY") && strings.HasPrefix(s, "SAIA_") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check(tt.got) {
				t.Errorf("%s = %q, failed check", tt.name, tt.got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Extension manifest
// ---------------------------------------------------------------------------

func TestExtensionManifest(t *testing.T) {
	b, err := os.ReadFile("../extension.json")
	if err != nil {
		t.Skip("extension.json not found")
	}
	var ext struct {
		Name    string `json:"name"`
		Exec    string `json:"exec"`
		Enabled bool   `json:"enabled"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ext.Name != "zot-saia-plugin" {
		t.Errorf("name = %q", ext.Name)
	}
	if ext.Exec != "./zot-saia-plugin" {
		t.Errorf("exec = %q", ext.Exec)
	}
	if !ext.Enabled {
		t.Error("expected enabled=true")
	}
}

// ---------------------------------------------------------------------------
// Skill file
// ---------------------------------------------------------------------------

func TestSkillFile(t *testing.T) {
	b, err := os.ReadFile("../skills/saia-models.md")
	if err != nil {
		t.Skip("skill file not found")
	}
	content := string(b)
	required := []string{"## Available Models", "saia/glm-4.7", "SAIA_API_KEY", "## Usage"}
	for _, s := range required {
		if !strings.Contains(content, s) {
			t.Errorf("skill file missing: %q", s)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration test: wire protocol lifecycle
// ---------------------------------------------------------------------------

func TestWireProtocolLifecycle(t *testing.T) {
	bin := *binaryPath
	if bin == "" {
		candidates := []string{
			filepath.Join("..", "zot-saia-plugin"),
			filepath.Join("..", "..", "zot-saia-plugin"),
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.Mode()&0o111 != 0 {
				bin = c
				break
			}
		}
	}
	if bin == "" {
		t.Skip("skipping: build the binary first (./build.sh) or set -binary flag")
	}

	sess := wiretest.NewSession(t, bin)
	defer sess.Shutdown()

	hello := sess.Out[0]
	if name, _ := hello["name"].(string); name != "zot-saia-plugin" {
		t.Errorf("hello name = %q", name)
	}

	found := false
	for _, f := range sess.Out {
		if f["type"] == "register_command" {
			if name, _ := f["name"].(string); name == "saia-models" {
				found = true
			}
		}
	}
	if !found {
		t.Error("/saia-models command not registered")
	}

	resp := sess.InvokeCommand("saia-models", "")
	if action, _ := resp["action"].(string); action != "prompt" {
		t.Errorf("response action = %q, want prompt", action)
	}
	prompt, _ := resp["prompt"].(string)
	if !strings.Contains(prompt, "SAIA") {
		t.Error("prompt missing provider name")
	}
	if !strings.Contains(prompt, "saia/glm-4.7") {
		t.Error("prompt missing model")
	}
}
