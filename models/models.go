// Package models defines the SAIA Academic Cloud model catalog
// and provides prompt generation for the zot extension.
package models

import (
	"fmt"
	"strings"
)

// Model describes one model available on the SAIA Academic Cloud.
type Model struct {
	ID   string
	Name string
	Ctx  string
}

// All known models served by SAIA.
var All = []Model{
	{ID: "glm-4.7", Name: "GLM 4.7", Ctx: "128K"},
	{ID: "qwen3.5-397b-a17b", Name: "Qwen 3.5 397B", Ctx: "128K"},
	{ID: "qwen3.5-122b-a10b", Name: "Qwen 3.5 122B", Ctx: "128K"},
	{ID: "devstral-2-123b-instruct-2512", Name: "DevStral 2 123B", Ctx: "128K"},
	{ID: "openai-gpt-oss-120b", Name: "GPT-OSS 120B", Ctx: "128K"},
	{ID: "qwen3.6-27b", Name: "Qwen 3.6 27B", Ctx: "128K"},
}

// ProviderPrefix used in model IDs.
const ProviderPrefix = "saia"

// ProviderName is the human-readable provider name.
const ProviderName = "SAIA Academic Cloud"

// BaseURL is the OpenAI-compatible API endpoint.
const BaseURL = "https://chat-ai.academiccloud.de/v1"

// APIKeyEnv is the environment variable name for the API key.
const APIKeyEnv = "SAIA_API_KEY"

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
	b.WriteString("| Model ID | Name | Context | Reasoning |\n")
	b.WriteString("|----------|------|---------|----------|\n")
	for _, m := range models {
		fmt.Fprintf(&b, "| %s | %s | %s | ✅ |\n", m.FullID(), m.Name, m.Ctx)
	}
	b.WriteString("\n### Usage\n\n")
	b.WriteString("Switch to a SAIA model:\n  /model saia/glm-4.7\n\n")
	b.WriteString("Large reasoning model:\n  /model saia/qwen3.5-397b-a17b\n\n")
	b.WriteString("Lightweight model for quick tasks:\n  /model saia/qwen3.6-27b\n\n")
	b.WriteString("### API Key\n\n")
	b.WriteString("Set via environment variable:\n  export SAIA_API_KEY=\"your-key\"\n\n")
	b.WriteString("Or add to auth.json (saia key), or use /login.\n")
	return b.String()
}
