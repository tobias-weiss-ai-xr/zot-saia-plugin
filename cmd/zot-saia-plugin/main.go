package main

import (
	"fmt"

	"github.com/patriceckhart/zot/packages/agent/ext"
	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/cache"
	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/models"
)

func main() {
	e := ext.New("zot-saia-plugin", "1.2.0")

	// Auto-update: when the extension loads, attempt a (cached) refresh of
	// the model catalog from the live SAIA API. On a cache hit this is a
	// no-op; with SAIA_API_KEY set it also picks up newly added/removed
	// models so the static snapshot stays current.
	e.OnHello(func(h ext.HostInfo) {
		res, err := models.RefreshModelCatalog("", false)
		if err != nil {
			e.Logf("auto-refresh skipped: %v", err)
			return
		}
		if res.Changed {
			e.Logf("model catalog auto-updated: %s", res.Summary())
			if res.Added != nil || res.Removed != nil {
				e.Notify("info", fmt.Sprintf("SAIA models updated: %s", res.Summary()))
			}
		}
	})

	e.Command("saia-models", "list SAIA Academic Cloud models and usage", func(args string) ext.Response {
		// Optional auto-update: /saia-models refresh
		if args == "refresh" || args == "sync" {
			res, err := models.RefreshModelCatalog("", true)
			if err != nil {
				return ext.Errorf("SAIA refresh failed: %v", err)
			}
			return ext.Prompt(fmt.Sprintf("SAIA models refreshed: %s\n\n%s", res.Summary(), models.Prompt()))
		}
		return ext.Prompt(models.Prompt())
	})

	e.Command("saia-sync", "refresh SAIA model catalog from the live API (cached)", func(args string) ext.Response {
		force := args == "force"
		res, err := models.RefreshModelCatalog("", force)
		if err != nil {
			return ext.Errorf("SAIA sync failed: %v", err)
		}
		if res.Changed {
			return ext.Prompt(fmt.Sprintf("SAIA model catalog updated: %s\n\nCurrent models:\n%s",
				res.Summary(), models.Prompt()))
		}
		return ext.Display(fmt.Sprintf("SAIA model catalog unchanged: %s", res.Summary()))
	})

	e.Command("saia-cache", "show or manage the SAIA L0/L1 cache", func(args string) ext.Response {
		switch args {
		case "clear":
			cache.Clear()
			return ext.Display("SAIA cache cleared (L0 + L1)")
		case "":
			s := cache.GetStats()
			return ext.Display(fmt.Sprintf(
				"SAIA cache:\n  L0 (memory): %s, %d entries, %d hits / %d misses, TTL %s\n  L1 (disk): %s, TTL %s%s",
				onOff(s.L0Enabled), s.L0Size, s.L0Hits, s.L0Misses, s.L0TTL,
				onOff(s.L1Enabled), s.L1TTL, dirOrEmpty(s.L1Dir),
			))
		default:
			return ext.Errorf("unknown arg %q — use 'clear' or nothing", args)
		}
	})

	if err := e.Run(); err != nil {
		e.Logf("fatal: %v", err)
	}
}

func onOff(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func dirOrEmpty(d string) string {
	if d == "" {
		return ""
	}
	return fmt.Sprintf("\n  L1 dir: %s", d)
}
