// Model auto-update: fetches the live SAIA model catalog from
// https://chat-ai.academiccloud.de/v1/models, keeps the in-repo
// snapshot (models.All / models.json) up to date, and notifies the user
// when the catalog changes.
//
// Like the pi/opencode plugins, model fetching is gated behind the
// optional L0/L1 cache (see the cache package): on a cache hit the local
// snapshot is used; on a miss the API is queried and the result is
// cached.
package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tobias-weiss-ai-xr/zot-saia-plugin/cache"
)

// ModelsEndpoint is the live SAIA model list endpoint.
// Declared as a var so tests can override it with an httptest server.
var ModelsEndpoint = BaseURL + "/models"

// lastRefresh tracks the last time the catalog was refreshed from the API.
var lastRefresh time.Time

// RefreshResult describes the outcome of an auto-update run.
type RefreshResult struct {
	Changed   bool     `json:"changed"`
	FromCache bool     `json:"from_cache"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Count     int      `json:"count"`
	Error     string   `json:"error,omitempty"`
}

// apiModel is the subset of the SAIA /models response we consume.
type apiModel struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type apiResponse struct {
	Data []apiModel `json:"data"`
}

// has reports whether s contains item.
func has(s []string, item string) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}

// RawIDSet returns the current set of model IDs in All.
func RawIDSet() map[string]bool {
	out := make(map[string]bool, len(All))
	for _, m := range All {
		out[m.ID] = true
	}
	return out
}

// mergeIntoAll updates the in-package All slice from a fresh catalog,
// preserving hand-authored Name/description details for models we
// already know and appending previously unknown models.
func mergeIntoAll(models []apiModel) RefreshResult {
	current := RawIDSet()
	var res RefreshResult
	seen := make(map[string]bool, len(models))

	// Append previously unknown, ready models.
	for _, m := range models {
		if m.Status != "ready" {
			continue
		}
		seen[m.ID] = true
		if current[m.ID] {
			continue
		}
		res.Added = append(res.Added, m.ID)
		All = append(All, Model{
			ID:         m.ID,
			Name:       friendlyName(m.ID),
			Ctx:        ctxFor(m.ID),
			MaxOut:     maxOutFor(m.ID),
			Reasoning:  has(m.Output, "thought"),
			Attachment: has(m.Input, "image") || has(m.Input, "image_url"),
			Category:   categorize(m.ID),
		})
	}

	// Drop models that are no longer on the API.
	var kept []Model
	for _, m := range All {
		if seen[m.ID] {
			kept = append(kept, m)
		} else {
			res.Removed = append(res.Removed, m.ID)
		}
	}
	All = kept

	sort.Strings(res.Added)
	sort.Strings(res.Removed)
	res.Changed = len(res.Added) > 0 || len(res.Removed) > 0
	res.Count = len(All)
	return res
}

// friendlyName converts an API model id into a display name.
func friendlyName(id string) string {
	// Prefer existing name if present in the json snapshot.
	if m := FindByID(id); m != nil && m.Name != "" {
		return m.Name
	}
	var b strings.Builder
	for _, part := range strings.Split(id, "-") {
		if part == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		r := []rune(part)
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		b.WriteString(string(r))
	}
	return b.String()
}

// categorize maps an id to one of the plugin's Category values.
func categorize(id string) Category {
	switch {
	case strings.Contains(id, "coder"):
		return CatCoder
	case strings.Contains(id, "medgemma"):
		return CatMedical
	case strings.Contains(id, "omni"):
		return CatVision
	case id == "openai-gpt-oss-120b":
		return CatLargeContext
	case id == "devstral-2-123b-instruct-2512" || id == "mistral-medium-3.5-128b" || id == "qwen3.6-35b-a3b":
		return CatAgentic
	case id == "qwen3.5-397b-a17b" || id == "qwen3.5-122b-a10b":
		return CatReasoning
	default:
		return CatGeneral
	}
}

// ctxFor maps an id to its context window label.
func ctxFor(id string) string {
	if id == "medgemma-27b-it" || id == "qwen3-omni-30b-a3b-instruct" {
		return "32K"
	}
	return "128K"
}

// maxOutFor maps an id to its max-output label.
func maxOutFor(id string) string {
	switch {
	case id == "qwen3.5-397b-a17b" || id == "qwen3.5-122b-a10b":
		return "32K"
	case id == "medgemma-27b-it" || id == "qwen3-omni-30b-a3b-instruct" || id == "meta-llama-3.1-8b-instruct":
		return "4K"
	default:
		return "16K"
	}
}

// fetchFromAPI hits the live SAIA /models endpoint. The API key is read
// from SAIA_API_KEY (or passed explicitly).
func fetchFromAPI(apiKey string) ([]apiModel, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, ModelsEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SAIA API returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("SAIA API returned invalid JSON: %w", err)
	}
	return parsed.Data, nil
}

// RefreshModelCatalog optionally refreshes the model snapshot from the
// live SAIA API, using the L0/L1 cache. When apiKey is empty the
// SAIA_API_KEY environment variable is used.
//
// Returns a RefreshResult; on a cache hit (or when nothing changed) the
// result records that fact. Errors are returned for network/auth issues.
func RefreshModelCatalog(apiKey string, force bool) (RefreshResult, error) {
	key := apiKey
	if key == "" {
		key = os.Getenv(APIKeyEnv)
	}
	if key == "" {
		return RefreshResult{}, fmt.Errorf("no SAIA API key: set %s or pass one explicitly", APIKeyEnv)
	}

	const cacheKey = "models"
	fresh := func() ([]apiModel, error) {
		return fetchFromAPI(key)
	}

	data, fromCache, err := cache.Fetch(cacheKey, fresh, func(models []apiModel) bool {
		return len(models) > 0
	})
	if err != nil {
		return RefreshResult{Error: err.Error()}, err
	}

	res := mergeIntoAll(data)
	res.FromCache = fromCache
	lastRefresh = time.Now()
	return res, nil
}

// LastRefresh returns when the catalog was last updated from the API.
func LastRefresh() time.Time { return lastRefresh }

// Summary returns a one-line human-readable summary of a refresh result.
func (r RefreshResult) Summary() string {
	var parts []string
	if r.FromCache {
		parts = append(parts, "cached")
	}
	parts = append(parts, fmt.Sprintf("%d models", r.Count))
	if len(r.Added) > 0 {
		parts = append(parts, fmt.Sprintf("+%d added", len(r.Added)))
	}
	if len(r.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("-%d removed", len(r.Removed)))
	}
	if r.Error != "" {
		parts = append(parts, "error: "+r.Error)
	}
	return strings.Join(parts, ", ")
}
