// Package cache provides optional L0 (in-memory) + L1 (disk) caching
// for the SAIA plugin, mirroring the opencode/pi caching behaviour.
//
// Environment variables:
//   - SAIA_CACHE_L0=false  disable L0 (in-memory) caching
//   - SAIA_CACHE_L1=false  disable L1 (disk) caching
//
// Priority on a fetch: L0 → L1 → fresh network call. A L1 hit primes L0.
package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Default time-to-live durations.
const (
	L0_TTL = 5 * time.Minute  // short-lived in-memory
	L1_TTL = 24 * time.Hour   // persistent on disk
)

// Default L1 cache directory: ~/.cache/saia (matches pi/opencode plugins).
var defaultDir = filepath.Join(mustHomeDir(), ".cache", "saia")

// Stats describes current cache state.
type Stats struct {
	L0Enabled bool      `json:"l0_enabled"`
	L0Size    int       `json:"l0_size"`
	L0Hits    uint64    `json:"l0_hits"`
	L0Misses  uint64    `json:"l0_misses"`
	L1Enabled bool      `json:"l1_enabled"`
	L1Dir     string    `json:"l1_dir"`
	L0TTL     string    `json:"l0_ttl"`
	L1TTL     string    `json:"l1_ttl"`
	LastWrite time.Time `json:"last_write,omitempty"`
}

func mustHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return home
}

// l0Enabled reports whether L0 (in-memory) caching is enabled.
func l0Enabled() bool { return os.Getenv("SAIA_CACHE_L0") != "false" }

// l1Enabled reports whether L1 (disk) caching is enabled.
func l1Enabled() bool { return os.Getenv("SAIA_CACHE_L1") != "false" }

// SetDir overrides the L1 cache directory (used by tests).
func SetDir(dir string) { defaultDir = dir }

// l1Path resolves the on-disk file for a cache key.
func l1Path(key string) string {
	// Sanitize key into a safe filename.
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '_'
		}
	}, key)
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(defaultDir, safe+".json")
}

// ---------------------------------------------------------------------------
// L0: in-memory cache
// ---------------------------------------------------------------------------

type l0Entry struct {
	data      any
	timestamp time.Time
}

var (
	l0Mu    sync.RWMutex
	l0Cache = map[string]l0Entry{}
	l0Hits  uint64
	l0Miss  uint64
)

func getL0(key string) (any, bool) {
	l0Mu.RLock()
	e, ok := l0Cache[key]
	if !ok {
		l0Miss++
		l0Mu.RUnlock()
		return nil, false
	}
	if time.Since(e.timestamp) > L0_TTL {
		l0Miss++
		l0Mu.RUnlock()
		l0Mu.Lock()
		delete(l0Cache, key)
		l0Mu.Unlock()
		return nil, false
	}
	l0Hits++
	l0Mu.RUnlock()
	return e.data, true
}

func setL0(key string, data any) {
	l0Mu.Lock()
	defer l0Mu.Unlock()
	l0Cache[key] = l0Entry{data: data, timestamp: time.Now()}
}

// ---------------------------------------------------------------------------
// L1: disk cache
// ---------------------------------------------------------------------------

// l1Entry stores the JSON-encoded payload plus a timestamp.
// Raw is kept as bytes so callers can decode directly into their own
// concrete type (T) — decoding into `any` would lose concrete typing
// (e.g. numbers become float64, structs become maps).
type l1Entry struct {
	Raw       json.RawMessage `json:"raw"`
	Timestamp time.Time       `json:"timestamp"`
}

func getL1Raw(key string) ([]byte, bool) {
	b, err := os.ReadFile(l1Path(key))
	if err != nil {
		return nil, false
	}
	var e l1Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, false
	}
	if time.Since(e.Timestamp) > L1_TTL {
		_ = os.Remove(l1Path(key))
		return nil, false
	}
	return e.Raw, true
}

func setL1(key string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	e := l1Entry{Raw: raw, Timestamp: time.Now()}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		return
	}
	tmp := l1Path(key) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, l1Path(key))
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Fetch performs a cached fetch with optional validation.
//
//   - key: unique cache key for this data
//   - fresh: function producing fresh data (called only on cache miss)
//   - isValid: optional validator for cached payloads (nil = accept anything)
//
// Returns data, whether it came from cache, and any error from the fresh call.
func Fetch[T any](key string, fresh func() (T, error), isValid func(T) bool) (T, bool, error) {
	var zero T

	// L0 check
	if l0Enabled() {
		if v, ok := getL0(key); ok {
			if d, ok := v.(T); ok {
				return d, true, nil
			}
		}
	}

	// L1 check
	if l1Enabled() {
		if raw, ok := getL1Raw(key); ok {
			var d T
			if err := json.Unmarshal(raw, &d); err == nil && (isValid == nil || isValid(d)) {
				if l0Enabled() {
					setL0(key, d)
				}
				return d, true, nil
			}
		}
	}

	// Fresh fetch
	data, err := fresh()
	if err != nil {
		return zero, false, err
	}

	if l0Enabled() {
		setL0(key, data)
	}
	if l1Enabled() {
		setL1(key, data)
	}
	return data, false, nil
}

// Clear empties both L0 and L1 caches.
func Clear() {
	l0Mu.Lock()
	l0Cache = map[string]l0Entry{}
	l0Hits, l0Miss = 0, 0
	l0Mu.Unlock()

	// Best-effort remove of L1 files for this process's keys is not
	// possible without enumeration; remove the whole saia cache dir.
	if l1Enabled() {
		_ = os.RemoveAll(defaultDir)
	}
}

// ClearL1 removes only the L1 disk cache directory.
func ClearL1() {
	if l1Enabled() {
		_ = os.RemoveAll(defaultDir)
	}
}

// GetStats returns a snapshot of cache state (safe for reporting).
func GetStats() Stats {
	l0Mu.RLock()
	size := len(l0Cache)
	hits, misses := l0Hits, l0Miss
	l0Mu.RUnlock()
	return Stats{
		L0Enabled: l0Enabled(),
		L0Size:    size,
		L0Hits:    hits,
		L0Misses:  misses,
		L1Enabled: l1Enabled(),
		L1Dir:     defaultDir,
		L0TTL:     L0_TTL.String(),
		L1TTL:     L1_TTL.String(),
	}
}

// ErrDisabled is returned by ForceRefresh when a fetch is required but
// both cache tiers are disabled (used to gate refresh attempts).
var ErrDisabled = errors.New("SAIA caching disabled (SAIA_CACHE_L0=false and SAIA_CACHE_L1=false)")
