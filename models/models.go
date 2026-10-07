// Package models defines the SAIA Academic Cloud model catalog
// and provides prompt generation for the zot extension.
package models

import (
	"fmt"
	"strings"
)

// Category classifies a model's primary use case.
type Category string

const (
	CatReasoning    Category = "reasoning"
	CatCoder        Category = "coder"
	CatAgentic      Category = "agentic"
	CatVision       Category = "vision"
	CatMedical      Category = "medical"
	CatLargeContext Category = "large-context"
	CatGeneral      Category = "general"
)

// Model describes one model available on the SAIA Academic Cloud.
type Model struct {
	ID         string
	Name       string
	Ctx        string
	MaxOut     string
	CtxTokens  int // exact context-window tokens (from data/saia-models.json)
	MaxTokens  int // exact max-output tokens (from data/saia-models.json)
	Reasoning  bool
	Attachment bool
	Category   Category
}

// All known models served by SAIA (synced with live API 2025-08-22).
// All known models served by SAIA, generated from data/saia-models.json
// (the canonical collected facts; models_test.go guards drift from facts).
var All = []Model{
	// Reasoning (output includes "thought")
	{ID: "deepseek-v4-flash-0731", Name: "DeepSeek V4 Flash 0731", Ctx: "1M", CtxTokens: 1000000, MaxOut: "32K", MaxTokens: 32768, Reasoning: true, Category: CatGeneral},
	{ID: "glm-5.3-flash", Name: "GLM 5.3 Flash", Ctx: "1M", CtxTokens: 1000000, MaxOut: "32K", MaxTokens: 32768, Reasoning: true, Attachment: true, Category: CatAgentic},
	{ID: "qwen3.5-397b-a17b", Name: "Qwen 3.5 397B A17B", Ctx: "256K", CtxTokens: 256000, MaxOut: "32K", MaxTokens: 32768, Reasoning: true, Attachment: true, Category: CatReasoning},
	{ID: "qwen3.6-35b-a3b", Name: "Qwen 3.6 35B A3B", Ctx: "262K", CtxTokens: 262000, MaxOut: "16K", MaxTokens: 16384, Reasoning: true, Attachment: true, Category: CatAgentic},
	{ID: "qwen3.8-27b", Name: "Qwen 3.8 27B", Ctx: "262K", CtxTokens: 262000, MaxOut: "32K", MaxTokens: 32768, Reasoning: true, Category: CatReasoning},

	// Coder
	{ID: "qwen3-coder-next", Name: "Qwen 3 Coder Next", Ctx: "256K", CtxTokens: 256000, MaxOut: "16K", MaxTokens: 16384, Category: CatCoder},

	// Agentic
	{ID: "devstral-2-123b-instruct-2512", Name: "DevStral 2 123B", Ctx: "256K", CtxTokens: 256000, MaxOut: "16K", MaxTokens: 16384, Category: CatAgentic},
	{ID: "mistral-medium-3.5-128b", Name: "Mistral Medium 3.5 128B", Ctx: "256K", CtxTokens: 256000, MaxOut: "8K", MaxTokens: 8192, Category: CatAgentic},

	// Large Context
	{ID: "openai-gpt-oss-120b", Name: "GPT-OSS 120B", Ctx: "128K", CtxTokens: 128000, MaxOut: "8K", MaxTokens: 8192, Reasoning: true, Category: CatLargeContext},

	// Vision (image/audio input)
	{ID: "qwen3-omni-30b-a3b-instruct", Name: "Qwen 3 Omni 30B", Ctx: "256K", CtxTokens: 256000, MaxOut: "4K", MaxTokens: 4096, Attachment: true, Category: CatVision},

	// General
	{ID: "gemma-4-31b-it", Name: "Gemma 4 31B", Ctx: "256K", CtxTokens: 256000, MaxOut: "8K", MaxTokens: 8192, Attachment: true, Category: CatGeneral},
	{ID: "qwen3-30b-a3b-instruct-2507", Name: "Qwen 3 30B A3B", Ctx: "256K", CtxTokens: 256000, MaxOut: "16K", MaxTokens: 16384, Category: CatGeneral},
	{ID: "apertus-70b-instruct-2509", Name: "Apertus 70B", Ctx: "65K", CtxTokens: 65000, MaxOut: "8K", MaxTokens: 8192, Category: CatGeneral},
	{ID: "meta-llama-3.1-8b-instruct", Name: "Llama 3.1 8B", Ctx: "128K", CtxTokens: 128000, MaxOut: "4K", MaxTokens: 4096, Category: CatGeneral},
}

// ProviderPrefix used in model IDs.
const ProviderPrefix = "saia"

// ProviderName is the human-readable provider name.
const ProviderName = "SAIA Academic Cloud"

// BaseURL is the OpenAI-compatible API endpoint.
const BaseURL = "https://chat-ai.academiccloud.de/v1"

// APIKeyEnv is the environment variable name for the API key.
const APIKeyEnv = "SAIA_API_KEY"

// ReasoningEmoji returns ✅ or ❌ for reasoning capability.
func ReasoningEmoji(r bool) string {
	if r {
		return "✅"
	}
	return "—"
}

// AttachmentEmoji returns a 🖼 symbol when a model supports image/audio input.
func AttachmentEmoji(a bool) string {
	if a {
		return "🖼"
	}
	return "—"
}

// FindByID returns the model with the given ID, or nil.
func FindByID(id string) *Model {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}

// FindByName returns the first model matching the given name, or nil.
func FindByName(name string) *Model {
	for i := range All {
		if All[i].Name == name {
			return &All[i]
		}
	}
	return nil
}

// FilterByCategory returns models matching the given category.
func FilterByCategory(cat Category) []Model {
	var out []Model
	for _, m := range All {
		if m.Category == cat {
			out = append(out, m)
		}
	}
	return out
}

// OutModel is the JSON shape zot's models.json schema expects for a provider
// model (packages/provider/usermodels.go UserModel). The plugin exports the
// catalog in this form so /saia-install-models can write $ZOT_HOME/models.json
// without any manual or host-dependent serialization.
type OutModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextWindow int    `json:"contextWindow"`
	MaxTokens     int    `json:"maxTokens"`
	Reasoning     bool   `json:"reasoning"`
}

// parseTokens converts a shorthand label ("256K", "1M") to a count. It is
// retained for one-way token-count conversions in tests/docs.
func parseTokens(label string) int {
	if len(label) == 0 {
		return 0
	}
	mult := 1
	last := label[len(label)-1]
	switch last {
	case 'K':
		mult = 1000
		label = label[:len(label)-1]
	case 'M':
		mult = 1000000
		label = label[:len(label)-1]
	}
	n := 0
	for _, r := range label {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n * mult
}

// OutModels returns the full catalog serialized for a models.json
// "providers.<name>.models" block, in catalog order so the first entry is the
// provider default (deepseek-v4-flash-0731). Exact token counts come from the
// Model.CtxTokens/MaxTokens fields (source: data/saia-models.json), not the
// rounded display labels.
func OutModels() []OutModel {
	out := make([]OutModel, 0, len(All))
	for _, m := range All {
		out = append(out, OutModel{
			ID:            m.ID,
			Name:          m.Name,
			ContextWindow: m.CtxTokens,
			MaxTokens:     m.MaxTokens,
			Reasoning:     m.Reasoning,
		})
	}
	return out
}

// FullID returns the provider-prefixed model ID (e.g. "saia/glm-5.3-flash").
func (m Model) FullID() string {
	return ProviderPrefix + "/" + m.ID
}

// Prompt builds the markdown-formatted model reference prompt
// shown when the user runs /saia-models.
func Prompt() string {
	return BuildPrompt(All)
}

// BuildPrompt generates a markdown model reference table from any model slice.
func BuildPrompt(models []Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s — Available Models\n\n", ProviderName)
	b.WriteString("| Model ID | Name | Ctx | Out | Reasoning | Attach | Category |\n")
	b.WriteString("|----------|------|-----|-----|-----------|--------|----------|\n")
	for _, m := range models {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			m.FullID(), m.Name, m.Ctx, m.MaxOut,
			ReasoningEmoji(m.Reasoning), AttachmentEmoji(m.Attachment),
			string(m.Category))
	}
	b.WriteString("\n### Quick Switch\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Best for agentic coding\n")
	b.WriteString("  /model saia/glm-5.3-flash\n\n")
	b.WriteString("# Flagship reasoning (long context)\n")
	b.WriteString("  /model saia/qwen3.5-397b-a17b\n\n")
	b.WriteString("# Code-specialized\n")
	b.WriteString("  /model saia/qwen3-coder-next\n\n")
	b.WriteString("# Fast & lightweight\n")
	b.WriteString("  /model saia/deepseek-v4-flash-0731\n\n")
	b.WriteString("# Vision (image input)\n")
	b.WriteString("  /model saia/qwen3.6-35b-a3b\n\n")
	b.WriteString("# Budget (cheapest)\n")
	b.WriteString("  /model saia/meta-llama-3.1-8b-instruct\n")
	b.WriteString("```\n\n")
	b.WriteString("### Rate Limits\n\n")
	b.WriteString("SAIA enforces: **30 req/min · 200/hour · 1,000/day · 3,000/month**\n")
	b.WriteString("Check remaining quota at [SAIA dashboard](https://chat-ai.academiccloud.de).\n\n")
	b.WriteString("### API Key\n\n")
	b.WriteString("Set via environment variable:\n  export SAIA_API_KEY=\"your-key\"\n\n")
	b.WriteString("Or add to auth.json (saia key), or use /login.\n")
	return b.String()
}
