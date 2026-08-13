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
	Reasoning  bool
	Attachment bool
	Category   Category
}

// All known models served by SAIA (synced with live API).
var All = []Model{
	// Reasoning
	{ID: "qwen3.5-397b-a17b", Name: "Qwen 3.5 397B", Ctx: "128K", MaxOut: "32K", Reasoning: true, Attachment: true, Category: CatReasoning},
	{ID: "qwen3.5-122b-a10b", Name: "Qwen 3.5 122B", Ctx: "128K", MaxOut: "32K", Reasoning: true, Attachment: true, Category: CatReasoning},
	{ID: "glm-4.7", Name: "GLM 4.7", Ctx: "128K", MaxOut: "16K", Reasoning: false, Category: CatGeneral},
	{ID: "qwen3-30b-a3b-instruct-2507", Name: "Qwen 3 30B", Ctx: "128K", MaxOut: "16K", Reasoning: true, Category: CatReasoning},

	// Coder
	{ID: "qwen3-coder-next", Name: "Qwen 3 Coder Next", Ctx: "128K", MaxOut: "16K", Category: CatCoder},

	// Agentic
	{ID: "devstral-2-123b-instruct-2512", Name: "DevStral 2 123B", Ctx: "128K", MaxOut: "16K", Reasoning: false, Category: CatAgentic},
	{ID: "mistral-medium-3.5-128b", Name: "Mistral Medium 3.5 128B", Ctx: "128K", MaxOut: "8K", Category: CatAgentic},
	{ID: "qwen3.6-35b-a3b", Name: "Qwen 3.6 35B", Ctx: "128K", MaxOut: "16K", Attachment: true, Category: CatAgentic},

	// Large Context
	{ID: "openai-gpt-oss-120b", Name: "GPT-OSS 120B", Ctx: "128K", MaxOut: "8K", Category: CatLargeContext},

	// Medical
	{ID: "medgemma-27b-it", Name: "MedGemma 27B", Ctx: "32K", MaxOut: "4K", Attachment: true, Category: CatMedical},

	// Vision
	{ID: "qwen3-omni-30b-a3b-instruct", Name: "Qwen 3 Omni 30B", Ctx: "32K", MaxOut: "4K", Attachment: true, Category: CatVision},

	// General
	{ID: "deepseek-v4-flash-0731", Name: "DeepSeek V4 Flash", Ctx: "128K", MaxOut: "16K", Category: CatGeneral},
	{ID: "qwen3.6-27b", Name: "Qwen 3.6 27B", Ctx: "128K", MaxOut: "16K", Category: CatGeneral},
	{ID: "gemma-4-31b-it", Name: "Gemma 4 31B", Ctx: "128K", MaxOut: "8K", Attachment: true, Category: CatGeneral},
	{ID: "apertus-70b-instruct-2509", Name: "Apertus 70B", Ctx: "128K", MaxOut: "8K", Category: CatGeneral},
	{ID: "meta-llama-3.1-8b-instruct", Name: "Meta Llama 3.1 8B", Ctx: "128K", MaxOut: "4K", Category: CatGeneral},
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

// AttachmentEmoji returns 🖼 or — for attachment capability.
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

// FullID returns the provider-prefixed model ID (e.g. "saia/glm-4.7").
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
	b.WriteString("  /model saia/glm-4.7\n\n")
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
