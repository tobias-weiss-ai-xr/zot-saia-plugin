package models

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/cache"
)

// saveAll saves the current global All slice and restores it after tests.
func saveAll() []Model {
	orig := All
	return orig
}

func restoreAll(orig []Model) {
	All = orig
}

func TestMergeIntoAll_AddsNewModels(t *testing.T) {
	orig := saveAll()
	defer restoreAll(orig)

	All = []Model{{ID: "existing-1", Name: "Existing", Ctx: "128K", MaxOut: "16K", Category: CatGeneral}}

	in := []apiModel{
		{ID: "existing-1", Status: "ready", Input: []string{"text"}, Output: []string{"text"}},
		{ID: "brand-new", Status: "ready", Input: []string{"text", "image"}, Output: []string{"text", "thought"}},
	}
	res := mergeIntoAll(in)

	if !res.Changed {
		t.Error("expected Changed=true when adding a model")
	}
	if len(res.Added) != 1 || res.Added[0] != "brand-new" {
		t.Errorf("Added = %v, want [brand-new]", res.Added)
	}
	found := FindByID("brand-new")
	if found == nil {
		t.Fatal("FindByID(brand-new) = nil")
	}
	if !found.Reasoning || !found.Attachment {
		t.Errorf("new model reasoning=%v attachment=%v, want true/true", found.Reasoning, found.Attachment)
	}
}

func TestMergeIntoAll_RemovesStaleModels(t *testing.T) {
	orig := saveAll()
	defer restoreAll(orig)

	All = []Model{
		{ID: "still-here", Name: "Still", Ctx: "128K", MaxOut: "16K", Category: CatGeneral},
		{ID: "gone-away", Name: "Gone", Ctx: "128K", MaxOut: "16K", Category: CatGeneral},
	}

	in := []apiModel{{ID: "still-here", Status: "ready"}}
	res := mergeIntoAll(in)

	if !res.Changed {
		t.Error("expected Changed=true when removing a model")
	}
	if len(res.Removed) != 1 || res.Removed[0] != "gone-away" {
		t.Errorf("Removed = %v, want [gone-away]", res.Removed)
	}
	if FindByID("gone-away") != nil {
		t.Error("gone-away should have been removed")
	}
	if FindByID("still-here") == nil {
		t.Error("still-here should remain")
	}
}

func TestMergeIntoAll_SkipsNonReady(t *testing.T) {
	orig := saveAll()
	defer restoreAll(orig)

	All = nil
	in := []apiModel{
		{ID: "ready-model", Status: "ready"},
		{ID: "loading-model", Status: "loading"},
	}
	mergeIntoAll(in)

	if FindByID("loading-model") != nil {
		t.Error("loading model should not be added")
	}
	if FindByID("ready-model") == nil {
		t.Error("ready model should be added")
	}
}

func TestCategorize(t *testing.T) {
	tests := []struct {
		id   string
		want Category
	}{
		{"qwen3-coder-next", CatCoder},
		{"medgemma-27b-it", CatMedical},
		{"qwen3-omni-30b-a3b-instruct", CatVision},
		{"openai-gpt-oss-120b", CatLargeContext},
		{"devstral-2-123b-instruct-2512", CatAgentic},
		{"qwen3.5-397b-a17b", CatReasoning},
		{"glm-4.7", CatGeneral},
		{"totally-unknown", CatGeneral},
	}
	for _, tt := range tests {
		if got := categorize(tt.id); got != tt.want {
			t.Errorf("categorize(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestHas(t *testing.T) {
	if !has([]string{"text", "thought"}, "thought") {
		t.Error("expected has() to find thought")
	}
	if has([]string{"text"}, "thought") {
		t.Error("expected has() not to find thought")
	}
}

// startTestServer serves a canned SAIA /models response and records the
// Authorization header it received.
func startTestServer(t *testing.T, body string, recordAuth func(string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recordAuth != nil {
			recordAuth(r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRefreshModelCatalog_CachesAndFetches(t *testing.T) {
	orig := saveAll()
	defer restoreAll(orig)
	cacheDir := t.TempDir()
	cache.SetDir(cacheDir)
	defer cache.Clear()

	// Save original endpoint so we can restore it after the test.
	origEndpoint := ModelsEndpoint

	body := `{"data":[{"id":"model-a","status":"ready","input":["text"],"output":["text"]},{"id":"model-b","status":"ready","input":["text"],"output":["text","thought"]}]}`
	srv := startTestServer(t, body, nil)
	ModelsEndpoint = srv.URL
	t.Cleanup(func() { ModelsEndpoint = origEndpoint })

	t.Setenv("SAIA_API_KEY", "test-key")
	All = nil

	res, err := RefreshModelCatalog("", false)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if res.Count != 2 {
		t.Errorf("Count = %d, want 2", res.Count)
	}
	if len(res.Added) != 2 {
		t.Errorf("Added = %v, want both models", res.Added)
	}

	foundB := FindByID("model-b")
	if foundB == nil || !foundB.Reasoning {
		t.Error("model-b should have Reasoning=true (thought output)")
	}

	// A second call should hit the cache and not change anything.
	res2, err := RefreshModelCatalog("", false)
	if err != nil {
		t.Fatalf("refresh2: %v", err)
	}
	if !res2.FromCache {
		t.Error("second refresh should come from cache")
	}
	if res2.Changed {
		t.Error("second refresh should not report changes")
	}
}

func TestRefreshModelCatalog_NoKeyError(t *testing.T) {
	// Ensure no key is set.
	t.Setenv("SAIA_API_KEY", "")
	_, err := RefreshModelCatalog("", false)
	if err == nil {
		t.Fatal("expected error when no API key")
	}
	if !strings.Contains(err.Error(), "SAIA_API_KEY") {
		t.Errorf("error should mention SAIA_API_KEY: %v", err)
	}
}

func TestRefreshModelCatalog_AuthHeaderSent(t *testing.T) {
	orig := saveAll()
	defer restoreAll(orig)

	cacheDir := t.TempDir()
	cache.SetDir(cacheDir)
	defer cache.Clear()
	origEndpoint := ModelsEndpoint
	defer func() { ModelsEndpoint = origEndpoint }()

	srv := startTestServer(t, `{"data":[]}`, func(auth string) {
		if auth != "Bearer my-secret" {
			t.Errorf("Authorization = %q, want Bearer my-secret", auth)
		}
	})
	ModelsEndpoint = srv.URL

	_, _ = RefreshModelCatalog("my-secret", false)
}

func TestRefreshSummary(t *testing.T) {
	r := RefreshResult{Added: []string{"a"}, Removed: []string{"b"}, Count: 3}
	s := r.Summary()
	if !strings.Contains(s, "3 models") || !strings.Contains(s, "+1 added") || !strings.Contains(s, "-1 removed") {
		t.Errorf("summary = %q", s)
	}

	r2 := RefreshResult{FromCache: true, Count: 16}
	if !strings.Contains(r2.Summary(), "cached") {
		t.Errorf("cached summary = %q", r2.Summary())
	}
}

// Ensure the module still builds and vet-cleans.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
