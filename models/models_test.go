package models_test

import (
	"encoding/json"
	"flag"
	"fmt"
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
	if got, want := len(models.All), 14; got != want {
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
		name       string
		id         string
		wantCtx    string
		wantMaxOut string
		wantName   string
		wantCat    models.Category
	}{
		{"deepseek", "deepseek-v4-flash-0731", "1M", "32K", "DeepSeek V4 Flash 0731", models.CatGeneral},
		{"glm53", "glm-5.3-flash", "1M", "32K", "GLM 5.3 Flash", models.CatAgentic},
		{"qwen397", "qwen3.5-397b-a17b", "256K", "32K", "Qwen 3.5 397B A17B", models.CatReasoning},
		{"qwen835", "qwen3.6-35b-a3b", "262K", "16K", "Qwen 3.6 35B A3B", models.CatAgentic},
		{"qwen827", "qwen3.8-27b", "262K", "32K", "Qwen 3.8 27B", models.CatReasoning},
		{"coder", "qwen3-coder-next", "256K", "16K", "Qwen 3 Coder Next", models.CatCoder},
		{"devstral", "devstral-2-123b-instruct-2512", "256K", "16K", "DevStral 2 123B", models.CatAgentic},
		{"mistral", "mistral-medium-3.5-128b", "256K", "8K", "Mistral Medium 3.5 128B", models.CatAgentic},
		{"gptoss", "openai-gpt-oss-120b", "128K", "8K", "GPT-OSS 120B", models.CatLargeContext},
		{"omni", "qwen3-omni-30b-a3b-instruct", "256K", "4K", "Qwen 3 Omni 30B", models.CatVision},
		{"gemma4", "gemma-4-31b-it", "256K", "8K", "Gemma 4 31B", models.CatGeneral},
		{"qwen30", "qwen3-30b-a3b-instruct-2507", "256K", "16K", "Qwen 3 30B A3B", models.CatGeneral},
		{"apertus", "apertus-70b-instruct-2509", "65K", "8K", "Apertus 70B", models.CatGeneral},
		{"llama8b", "meta-llama-3.1-8b-instruct", "128K", "4K", "Llama 3.1 8B", models.CatGeneral},
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
			if m.MaxOut != tt.wantMaxOut {
				t.Errorf("MaxOut = %q, want %q", m.MaxOut, tt.wantMaxOut)
			}
			if m.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", m.Name, tt.wantName)
			}
			if m.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", m.Category, tt.wantCat)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Facts-as-source guard: the shipped catalog must mirror data/saia-models.json
// ---------------------------------------------------------------------------

// TestCatalogMatchesFacts reads data/saia-models.json (the canonical collected
// facts) and asserts models.All agrees with it on model set, context window,
// max output, reasoning, attachment and category. This guarantees the Go
// catalog cannot silently drift from the live SAIA facts.
func TestCatalogMatchesFacts(t *testing.T) {
	b, err := os.ReadFile("../data/saia-models.json")
	if err != nil {
		t.Skip("data/saia-models.json not found")
	}
	var facts struct {
		Models []struct {
			ID        string `json:"id"`
			MaxOutput int    `json:"max_output"`
			Category  string `json:"category"`
			Reasoning *struct {
				Supported bool `json:"supported"`
			} `json:"reasoning"`
			Input         []string `json:"input"`
			ContextWindow struct {
				Tokens int64 `json:"tokens"`
			} `json:"context_window"`
		} `json:"models"`
	}
	if err := json.Unmarshal(b, &facts); err != nil {
		t.Fatalf("parse data/saia-models.json: %v", err)
	}

	byID := make(map[string]models.Model, len(models.All))
	for _, m := range models.All {
		byID[m.ID] = m
	}

	if got, want := len(models.All), len(facts.Models); got != want {
		t.Fatalf("All has %d models, facts have %d", got, want)
	}

	for _, f := range facts.Models {
		m, ok := byID[f.ID]
		if !ok {
			t.Errorf("model %q in facts but missing from models.All", f.ID)
			continue
		}
		if int64(modelCtxToInt(m.Ctx)) != f.ContextWindow.Tokens {
			t.Errorf("model %q: Ctx %q does not match facts ctx %d", f.ID, m.Ctx, f.ContextWindow.Tokens)
		}
		if m.MaxOut != maxOutLabel(f.MaxOutput) {
			t.Errorf("model %q: MaxOut %q does not match facts max_output %d", f.ID, m.MaxOut, f.MaxOutput)
		}
		wantReasoning := f.Reasoning != nil && f.Reasoning.Supported
		if m.Reasoning != wantReasoning {
			t.Errorf("model %q: Reasoning=%v, facts=%v", f.ID, m.Reasoning, wantReasoning)
		}
		wantAttach := attachmentFromInput(f.Input)
		if m.Attachment != wantAttach {
			t.Errorf("model %q: Attachment=%v, facts=%v", f.ID, m.Attachment, wantAttach)
		}
		if wantCat, ok := catFromString(f.Category); ok && m.Category != wantCat {
			t.Errorf("model %q: Category=%q, facts=%q", f.ID, m.Category, f.Category)
		}
	}
}

// modelCtxToInt converts a shorthand like "256K" / "1M" / "65K" back to
// token count for comparison with data/saia-models.json.
func modelCtxToInt(ctx string) int {
	if len(ctx) == 0 {
		return 0
	}
	mult := 1
	switch ctx[len(ctx)-1] {
	case 'K':
		mult = 1000
		ctx = ctx[:len(ctx)-1]
	case 'M':
		mult = 1000000
		ctx = ctx[:len(ctx)-1]
	default:
		mult = 1
	}
	n := 0
	for _, r := range ctx {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n * mult
}

// maxOutLabel formats a max-output token count as the "NNK" label.
func maxOutLabel(tokens int) string {
	return fmt.Sprintf("%dK", tokens/1000)
}

// attachmentFromInput reports whether a model accepts a non-text modality.
func attachmentFromInput(input []string) bool {
	for _, m := range input {
		if m != "text" {
			return true
		}
	}
	return false
}

// catFromString maps a fact category to a models.Category.
func catFromString(s string) (models.Category, bool) {
	switch s {
	case "reasoning":
		return models.CatReasoning, true
	case "coder":
		return models.CatCoder, true
	case "agentic":
		return models.CatAgentic, true
	case "vision":
		return models.CatVision, true
	case "medical":
		return models.CatMedical, true
	case "large-context":
		return models.CatLargeContext, true
	case "general":
		return models.CatGeneral, true
	default:
		return "", false
	}
}

// ---------------------------------------------------------------------------
// Attachment / Reasoning flags
// ---------------------------------------------------------------------------

func TestAttachmentFlags(t *testing.T) {
	attachModels := []string{
		"qwen3.5-397b-a17b", "qwen3.6-35b-a3b", "glm-5.3-flash",
		"gemma-4-31b-it", "qwen3-omni-30b-a3b-instruct",
	}
	for _, id := range attachModels {
		t.Run(id, func(t *testing.T) {
			m := models.FindByID(id)
			if m == nil {
				t.Fatalf("FindByID(%q) = nil", id)
			}
			if !m.Attachment {
				t.Errorf("expected Attachment=true for %q", id)
			}
		})
	}
}

func TestReasoningFlags(t *testing.T) {
	reasoningModels := []string{
		"qwen3.5-397b-a17b", "qwen3.6-35b-a3b", "glm-5.3-flash",
		"deepseek-v4-flash-0731", "qwen3.8-27b", "openai-gpt-oss-120b",
	}
	for _, id := range reasoningModels {
		t.Run(id, func(t *testing.T) {
			m := models.FindByID(id)
			if m == nil {
				t.Fatalf("FindByID(%q) = nil", id)
			}
			if !m.Reasoning {
				t.Errorf("expected Reasoning=true for %q", id)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FilterByCategory
// ---------------------------------------------------------------------------

func TestFilterByCategory(t *testing.T) {
	tests := []struct {
		cat      models.Category
		wantMin  int
		wantName string
	}{
		{models.CatReasoning, 2, "qwen3.5-397b-a17b"},
		{models.CatCoder, 1, "qwen3-coder-next"},
		{models.CatAgentic, 4, "devstral-2-123b-instruct-2512"},
		{models.CatVision, 1, "qwen3-omni-30b-a3b-instruct"},
		{models.CatLargeContext, 1, "openai-gpt-oss-120b"},
		{models.CatGeneral, 5, "deepseek-v4-flash-0731"},
	}
	for _, tt := range tests {
		t.Run(string(tt.cat), func(t *testing.T) {
			out := models.FilterByCategory(tt.cat)
			if len(out) < tt.wantMin {
				t.Errorf("FilterByCategory(%q) = %d models, want >= %d", tt.cat, len(out), tt.wantMin)
			}
			found := false
			for _, m := range out {
				if m.ID == tt.wantName {
					found = true
				}
			}
			if !found {
				t.Errorf("FilterByCategory(%q) missing %q", tt.cat, tt.wantName)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Emojis
// ---------------------------------------------------------------------------

func TestReasoningEmoji(t *testing.T) {
	if got := models.ReasoningEmoji(true); got != "✅" {
		t.Errorf("got %q", got)
	}
	if got := models.ReasoningEmoji(false); got != "—" {
		t.Errorf("got %q", got)
	}
}

func TestAttachmentEmoji(t *testing.T) {
	if got := models.AttachmentEmoji(true); got != "🖼" {
		t.Errorf("got %q", got)
	}
	if got := models.AttachmentEmoji(false); got != "—" {
		t.Errorf("got %q", got)
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
		{"found", "glm-5.3-flash", "GLM 5.3 Flash"},
		{"not found", "nonexistent", ""},
		{"empty string", "", ""},
		{"partial match", "glm", ""},
		{"deepseek-v4-flash", "deepseek-v4-flash-0731", "DeepSeek V4 Flash 0731"},
		{"qwen-coder-next", "qwen3-coder-next", "Qwen 3 Coder Next"},
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
		name      string
		search    string
		wantFound bool
		wantID    string
	}{
		{"found", "GLM 5.3 Flash", true, "glm-5.3-flash"},
		{"not found", "nonexistent", false, ""},
		{"empty", "", false, ""},
		{"deepseek", "DeepSeek V4 Flash 0731", true, "deepseek-v4-flash-0731"},
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
		{"single model", []models.Model{{ID: "x", Name: "X", Ctx: "64K", MaxOut: "8K"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.BuildPrompt(tt.models)
			if !strings.Contains(got, models.ProviderName) {
				t.Error("missing provider name")
			}
			if !strings.Contains(got, "### Quick Switch") {
				t.Error("missing quick switch section")
			}
			if !strings.Contains(got, "SAIA_API_KEY") {
				t.Error("missing API key env var")
			}
			if !strings.Contains(got, "Rate Limits") {
				t.Error("missing rate limits section")
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

	jsonIDs := make(map[string]bool, len(saia.Models))
	codeIDs := make(map[string]bool, len(models.All))
	for _, jm := range saia.Models {
		jsonIDs[jm.ID] = true
	}
	for _, cm := range models.All {
		codeIDs[cm.ID] = true
	}
	for id := range jsonIDs {
		if !codeIDs[id] {
			t.Errorf("model %q in models.json but not in code", id)
		}
	}
	for id := range codeIDs {
		if !jsonIDs[id] {
			t.Errorf("model %q in code but not in models.json", id)
		}
	}
}

// ---------------------------------------------------------------------------
// Virtual FS test
// ---------------------------------------------------------------------------

func TestModelsViaVirtualFS(t *testing.T) {
	content, err := json.Marshal(map[string]interface{}{
		"additional_providers": map[string]interface{}{
			"saia": map[string]interface{}{
				"api":         "openai-completions",
				"base_url":    models.BaseURL,
				"api_key_env": models.APIKeyEnv,
				"models": []map[string]interface{}{
					{"id": "deepseek-v4-flash-0731", "name": "DeepSeek V4 Flash 0731", "context_window": 131072},
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
	f.Add([]byte(`[{"id":"a","name":"A","ctx":"1K","max_out":"1K"}]`))
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
	required := []string{"## Available Models", "saia/glm-5.3-flash", "SAIA_API_KEY", "## Quick Switch", "deepseek-v4-flash-0731", "Rate Limits"}
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
	foundSync := false
	foundCache := false
	for _, f := range sess.Out {
		if f["type"] == "register_command" {
			switch name, _ := f["name"].(string); name {
			case "saia-models":
				found = true
			case "saia-sync":
				foundSync = true
			case "saia-cache":
				foundCache = true
			}
		}
	}
	if !found {
		t.Error("/saia-models command not registered")
	}
	if !foundSync {
		t.Error("/saia-sync command not registered")
	}
	if !foundCache {
		t.Error("/saia-cache command not registered")
	}

	resp := sess.InvokeCommand("saia-models", "")
	if action, _ := resp["action"].(string); action != "prompt" {
		t.Errorf("response action = %q, want prompt", action)
	}
	prompt, _ := resp["prompt"].(string)
	if !strings.Contains(prompt, "SAIA") {
		t.Error("prompt missing provider name")
	}
	if !strings.Contains(prompt, "saia/glm-5.3-flash") {
		t.Error("prompt missing model")
	}
	if !strings.Contains(prompt, "deepseek-v4-flash-0731") {
		t.Error("prompt missing deepseek-v4-flash-0731")
	}
	if !strings.Contains(prompt, "Rate Limits") {
		t.Error("prompt missing rate limits info (from PR #3)")
	}

	// /saia-cache should reply with cache status (display action).
	cacheResp := sess.InvokeCommand("saia-cache", "")
	if action, _ := cacheResp["action"].(string); action != "display" {
		t.Errorf("saia-cache response action = %q, want display", action)
	}
	cacheDisplay, _ := cacheResp["display"].(string)
	if !strings.Contains(cacheDisplay, "L0") || !strings.Contains(cacheDisplay, "L1") {
		t.Error("saia-cache display missing L0/L1 stats")
	}

	// /saia-cache clear should reset both tiers.
	clearResp := sess.InvokeCommand("saia-cache", "clear")
	if action, _ := clearResp["action"].(string); action != "display" {
		t.Errorf("saia-cache clear response action = %q, want display", action)
	}
	clearDisplay, _ := clearResp["display"].(string)
	if !strings.Contains(clearDisplay, "cleared") {
		t.Error("saia-cache clear should mention cleared")
	}
}
