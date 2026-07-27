package main

import (
	"fmt"
	"strings"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

// Models served by the SAIA Academic Cloud Hessen.
var models = []struct {
	ID   string
	Name string
	Ctx  string
}{
	{"glm-4.7", "GLM 4.7", "128K"},
	{"qwen3.5-397b-a17b", "Qwen 3.5 397B", "128K"},
	{"qwen3.5-122b-a10b", "Qwen 3.5 122B", "128K"},
	{"devstral-2-123b-instruct-2512", "DevStral 2 123B", "128K"},
	{"openai-gpt-oss-120b", "GPT-OSS 120B", "128K"},
	{"qwen3.6-27b", "Qwen 3.6 27B", "128K"},
}

func buildModelsPrompt() string {
	var b strings.Builder
	b.WriteString("## SAIA Academic Cloud — Available Models\n\n")
	b.WriteString("| Model ID | Name | Context | Reasoning |\n")
	b.WriteString("|----------|------|---------|----------|\n")
	for _, m := range models {
		fmt.Fprintf(&b, "| saia/%s | %s | %s | ✅ |\n", m.ID, m.Name, m.Ctx)
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

func main() {
	e := ext.New("zot-saia-plugin", "1.0.0")

	e.Command("saia-models", "list SAIA Academic Cloud models and usage", func(args string) ext.Response {
		return ext.Prompt(buildModelsPrompt())
	})

	if err := e.Run(); err != nil {
		e.Logf("fatal: %v", err)
	}
}
