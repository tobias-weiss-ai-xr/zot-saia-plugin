package main

import (
	"github.com/patriceckhart/zot/packages/agent/ext"
	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/models"
)

func main() {
	e := ext.New("zot-saia-plugin", "1.0.0")

	e.Command("saia-models", "list SAIA Academic Cloud models and usage", func(args string) ext.Response {
		return ext.Prompt(models.Prompt())
	})

	if err := e.Run(); err != nil {
		e.Logf("fatal: %v", err)
	}
}
