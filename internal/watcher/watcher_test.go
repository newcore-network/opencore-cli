package watcher

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestShouldIgnorePathForAutoloadArtifacts(t *testing.T) {
	w := &Watcher{}

	if !w.shouldIgnorePath(filepath.Join("resources", "sample-resource", ".opencore", "autoload.client.controllers.ts")) {
		t.Fatalf("expected .opencore autoload file to be ignored")
	}
}

func TestDebounceStateReschedulesAndTakesDuePaths(t *testing.T) {
	now := time.Unix(100, 0)
	state := make(debounceState)
	state.add("b.ts", now)
	state.add("a.ts", now.Add(100*time.Millisecond))
	state.add("b.ts", now.Add(200*time.Millisecond))

	if wait := state.next(now); wait != 600*time.Millisecond {
		t.Fatalf("next debounce = %v, want %v", wait, 600*time.Millisecond)
	}
	if due := state.takeDue(now.Add(650 * time.Millisecond)); len(due) != 1 || due[0] != "a.ts" {
		t.Fatalf("due paths = %v, want [a.ts]", due)
	}
	if due := state.takeDue(now.Add(700 * time.Millisecond)); len(due) != 1 || due[0] != "b.ts" {
		t.Fatalf("due paths = %v, want [b.ts]", due)
	}
}

func TestHandleLogsRejectsOversizedBody(t *testing.T) {
	w := &Watcher{logQueue: make(chan LogMessage, 1)}
	request := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewReader(make([]byte, maxLogBody+1)))
	response := httptest.NewRecorder()

	w.handleLogs(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandleLogsSanitizesTerminalControlCharacters(t *testing.T) {
	w := &Watcher{logQueue: make(chan LogMessage, 1)}
	body := `{"payload":[{"domain":"api\u001b[2J","message":"first\nsecond\r"}]}`
	request := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(body))
	response := httptest.NewRecorder()

	w.handleLogs(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	log := <-w.logQueue
	if strings.ContainsAny(log.Domain+log.Message, "\x1b\n\r") {
		t.Fatalf("control characters were not sanitized: domain=%q message=%q", log.Domain, log.Message)
	}
	if log.Domain != "api [2J" || log.Message != "first second " {
		t.Fatalf("unexpected sanitized log: domain=%q message=%q", log.Domain, log.Message)
	}
}

func TestSanitizeTerminalTextKeepsValidUTF8WhenTruncated(t *testing.T) {
	got := sanitizeTerminalText("abéé", 5)
	if got != "abé" {
		t.Fatalf("sanitized text = %q, want %q", got, "abé")
	}
}

func TestShouldIgnorePathForOutputAndDestination(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "build")
	destDir := filepath.Join(tmp, "deploy")
	w := &Watcher{
		config: &config.Config{
			OutDir:      outDir,
			Destination: destDir,
		},
	}

	if !w.shouldIgnorePath(filepath.Join(outDir, "resource", "core", "server.js")) {
		t.Fatalf("expected outDir path to be ignored")
	}
	if !w.shouldIgnorePath(filepath.Join(destDir, "resource", "core", "client.js")) {
		t.Fatalf("expected destination path to be ignored")
	}
}

func TestShouldIgnorePathDoesNotIgnoreSourceFiles(t *testing.T) {
	tmp := t.TempDir()
	w := &Watcher{
		config: &config.Config{
			OutDir:      filepath.Join(tmp, "build"),
			Destination: filepath.Join(tmp, "deploy"),
		},
	}

	if w.shouldIgnorePath(filepath.Join(tmp, "resources", "sample-resource", "src", "server", "main.ts")) {
		t.Fatalf("expected source file path to be watched")
	}
}
