// Package wiretest simulates the zot ↔ extension subprocess wire protocol
// (newline-delimited JSON over stdin/stdout) for integration testing.
package wiretest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Frame is a raw JSON object exchanged on the wire.
type Frame = map[string]any

// Session represents one host↔extension conversation.
type Session struct {
	Bin  string
	Cmd  *exec.Cmd
	In   *bytes.Buffer
	inWriter *os.File
	inCloser *os.File
	mu   sync.Mutex
	Out  []Frame
	Err  bytes.Buffer
	T    *testing.T
	Log  []string
	Done bool
}

// NewSession spawns the extension binary and completes the hello handshake.
func NewSession(t *testing.T, binPath string) *Session {
	t.Helper()
	s := &Session{T: t, Log: []string{}, Bin: binPath}

	s.Cmd = exec.Command(binPath)
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	s.Cmd.Stdin = inR
	s.inWriter = inW
	s.inCloser = inR
	s.Cmd.Stderr = &s.Err

	// Use StdoutPipe — the write side stays open until the child exits.
	pipe, err := s.Cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	dec := json.NewDecoder(pipe)

	if err := s.Cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", binPath, err)
	}

	// Read hello from extension.
	var hello Frame
	if err := dec.Decode(&hello); err != nil {
		t.Fatalf("read hello: %v (stderr: %s)", err, s.Err.String())
	}
	s.appendOut(hello)
	s.assertType(hello, "hello")

	// Send hello_ack.
	s.Send(&Frame{
		"type":             "hello_ack",
		"protocol_version": 1,
		"zot_version":      "0.3.13",
		"provider":         "test-provider",
		"model":            "test-model",
		"cwd":              t.TempDir(),
		"extension_dir":    t.TempDir(),
		"data_dir":         t.TempDir(),
	})

	// Drain registration frames + ready.
	for {
		var f Frame
		if err := dec.Decode(&f); err != nil {
			t.Fatalf("read frame after hello_ack: %v (stderr: %s)", err, s.Err.String())
		}
		s.appendOut(f)
		if f["type"] == "ready" {
			break
		}
	}

	// Background reader to drain remaining frames.
	go func() {
		for dec.More() {
			var f Frame
			if err := dec.Decode(&f); err != nil {
				return
			}
			s.appendOut(f)
		}
	}()

	return s
}

// appendOut adds a frame to the output slice (thread-safe).
func (s *Session) appendOut(f Frame) {
	s.mu.Lock()
	s.Out = append(s.Out, f)
	s.Log = append(s.Log, fmt.Sprintf("← %s", frameStr(f)))
	s.mu.Unlock()
}

// InvokeCommand sends a command_invoked frame and waits for a command_response.
func (s *Session) InvokeCommand(command, args string) Frame {
	s.T.Helper()
	id := fmt.Sprintf("test-%d", time.Now().UnixNano())
	s.Send(&Frame{
		"type": "command_invoked",
		"id":   id,
		"name": command,
		"args": args,
	})
	return s.waitFor("command_response", id, 5*time.Second)
}

// Shutdown gracefully terminates the extension.
func (s *Session) Shutdown() {
	s.T.Helper()
	if s.Done {
		return
	}
	s.Done = true
	s.Send(&Frame{"type": "shutdown"})
	s.inWriter.Close()
	s.inCloser.Close()
	if err := s.Cmd.Wait(); err != nil {
		s.T.Logf("wait: %v", err)
	}
}

// Send sends a raw frame to the extension.
func (s *Session) Send(f *Frame) {
	b, err := json.Marshal(f)
	if err != nil {
		s.T.Fatalf("marshal: %v", err)
	}
	b = append(b, '\n')
	if _, err := s.inWriter.Write(b); err != nil {
		s.T.Fatalf("write: %v", err)
	}
	s.mu.Lock()
	s.Log = append(s.Log, fmt.Sprintf("→ %s", string(b[:len(b)-1])))
	s.mu.Unlock()
}

// waitFor polls received frames until a matching type+id arrives.
func (s *Session) waitFor(wantType, wantID string, timeout time.Duration) Frame {
	s.T.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		for _, f := range s.Out {
			if f["type"] == wantType && f["id"] == wantID {
				s.mu.Unlock()
				return f
			}
		}
		s.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	s.T.Fatalf("timed out waiting for type=%q id=%q.\nTranscript:\n%s",
		wantType, wantID, strings.Join(s.Log, "\n"))
	return nil
}

// Transcript returns the full wire log for debugging.
func (s *Session) Transcript() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.Log, "\n")
}

func (s *Session) assertType(f Frame, want string) {
	s.T.Helper()
	got, ok := f["type"].(string)
	if !ok || got != want {
		s.T.Fatalf("frame type: got %q, want %q", got, want)
	}
}

func frameStr(f Frame) string {
	b, _ := json.Marshal(f)
	return string(b)
}

// Regex compiles a regex, failing the test on error.
func Regex(t *testing.T, pat string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(pat)
	if err != nil {
		t.Fatalf("bad regex %q: %v", pat, err)
	}
	return re
}

// ReadFile reads a file, failing the test on error.
func ReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// WriteFile writes a file, failing the test on error.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// Golden compares got against a golden file. If UPDATE_GOLDEN env is set, overwrites.
func Golden(t *testing.T, goldenPath string, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		WriteFile(t, goldenPath, got)
		t.Logf("updated golden: %s", goldenPath)
		return
	}
	want := ReadFile(t, goldenPath)
	if got != want {
		t.Errorf("golden mismatch:\n%s", diff(want, got))
	}
}

func diff(a, b string) string {
	aL, bL := strings.Split(a, "\n"), strings.Split(b, "\n")
	var sb strings.Builder
	max := len(aL)
	if len(bL) > max {
		max = len(bL)
	}
	for i := 0; i < max; i++ {
		al, bl := "", ""
		if i < len(aL) {
			al = aL[i]
		}
		if i < len(bL) {
			bl = bL[i]
		}
		if al != bl {
			fmt.Fprintf(&sb, "- %s\n+ %s\n", al, bl)
		}
	}
	return sb.String()
}
