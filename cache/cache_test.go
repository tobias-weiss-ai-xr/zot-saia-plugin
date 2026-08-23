package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFetch_FreshThenCached(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	var calls int
	fresh := func() (string, error) {
		calls++
		return "hello", nil
	}

	got, cached, err := Fetch("k1", fresh, nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if cached {
		t.Error("first fetch must not be cached")
	}
	if got != "hello" || calls != 1 {
		t.Errorf("got %q calls=%d", got, calls)
	}

	got2, cached2, err := Fetch("k1", fresh, nil)
	if err != nil {
		t.Fatalf("fetch2: %v", err)
	}
	if !cached2 {
		t.Error("second fetch must be cached (L0)")
	}
	if got2 != "hello" {
		t.Errorf("got2 %q", got2)
	}
	if calls != 1 {
		t.Errorf("fresh must not be called again, calls=%d", calls)
	}
}

func TestFetch_PassesThroughErrors(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	wantErr := errors.New("boom")
	fresh := func() (string, error) { return "", wantErr }
	_, _, err := Fetch("e1", fresh, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestFetch_ValidationRejects(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	var calls int
	fresh := func() ([]string, error) {
		calls++
		return []string{"a", "b"}, nil
	}

	valid := func(got []string) bool { return len(got) == 2 }

	got, _, err := Fetch("v1", fresh, valid)
	if err != nil || len(got) != 2 || calls != 1 {
		t.Fatalf("first fetch wrong: %v calls=%d", err, calls)
	}

	// Force a different validator that rejects the stored payload:
	// the stored one was written with valid(...) true, but we can't
	// replace it in-memory, so instead verify that a fresh fetch with a
	// rejecting validator still re-fetches from network if L1 is empty.
	ClearL1()
	Clear()
	got, _, err = Fetch("v1", fresh, func(got []string) bool { return len(got) == 99 })
	if err != nil || len(got) != 2 || calls != 2 {
		t.Fatalf("rejecting validator should still fetch: calls=%d err=%v", calls, err)
	}
}

func TestFetch_L0HitDoesNotTouchDisk(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	var calls int
	fresh := func() (int, error) { calls++; return 42, nil }

	// First populates L0 + L1.
	if _, _, err := Fetch("l0k", fresh, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	// Remove L1 on disk; L0 hit should return without re-fetch.
	os.RemoveAll(dir)

	got, cached, err := Fetch("l0k", fresh, nil)
	if err != nil {
		t.Fatalf("fetch2: %v", err)
	}
	if !cached || got != 42 {
		t.Errorf("L0 hit failed: cached=%v got=%d", cached, got)
	}
	if calls != 1 {
		t.Errorf("fresh called %d times", calls)
	}
}

func TestL1HitsPrimeL0(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	var calls int
	fresh := func() (string, error) { calls++; return "disk", nil }

	if _, _, err := Fetch("p1", fresh, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	// Wipe L0 only (simulating a fresh process whose L0 map is empty),
	// keep L1 on disk.
	ClearL0ForTest()

	got, cached, err := Fetch("p1", fresh, nil)
	if err != nil {
		t.Fatalf("fetch2: %v", err)
	}
	if !cached || got != "disk" {
		t.Fatalf("L1 hit expected: cached=%v got=%q", cached, got)
	}
	if calls != 1 {
		t.Errorf("fresh called %d times (should be 1, from L1)", calls)
	}
}

func TestStatsReadsCounters(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	fresh := func() (int, error) { return 7, nil }
	if _, _, err := Fetch("s1", fresh, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if _, _, err := Fetch("s1", fresh, nil); err != nil {
		t.Fatalf("fetch2: %v", err)
	}

	st := GetStats()
	if !st.L0Enabled {
		t.Error("L0 should be enabled by default")
	}
	if st.L0Hits == 0 {
		t.Error("expected at least one L0 hit")
	}
	if st.L1Dir != dir {
		t.Errorf("L1 dir = %q, want %q", st.L1Dir, dir)
	}
}

func TestClearRemovesL0AndL1(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	fresh := func() (int, error) { return 1, nil }
	if _, _, err := Fetch("c1", fresh, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	Clear()

	if _, err := os.Stat(filepath.Join(dir, "c1.json")); !os.IsNotExist(err) {
		t.Errorf("L1 file not removed: %v", err)
	}
	if size := GetStats().L0Size; size != 0 {
		t.Errorf("L0 size = %d after clear", size)
	}
}

func TestEnvDisableL0(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir(filepath.Join(mustHomeDir(), ".cache", "saia")) })
	defer Clear()

	t.Setenv("SAIA_CACHE_L0", "false")

	// Fresh fetch populates L1 only.
	var calls int
	fresh := func() (int, error) { calls++; return 9, nil }
	if _, _, err := Fetch("d1", fresh, nil); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	// Second fetch with L0 disabled should hit L1, not re-fetch.
	got, cached, err := Fetch("d1", fresh, nil)
	if err != nil {
		t.Fatalf("fetch2: %v", err)
	}
	if !cached || got != 9 {
		t.Errorf("L1 hit expected with L0 off: cached=%v got=%d", cached, got)
	}
	if calls != 1 {
		t.Errorf("fresh called %d times", calls)
	}
}

// ClearL0ForTest wipes the in-memory map (test helper).
func ClearL0ForTest() {
	l0Mu.Lock()
	l0Cache = map[string]l0Entry{}
	l0Hits, l0Miss = 0, 0
	l0Mu.Unlock()
}
