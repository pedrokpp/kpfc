package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNew_SelectsImplementation(t *testing.T) {
	if _, ok := New("json", false).(*JSONLogger); !ok {
		t.Fatal("expected JSONLogger for json format")
	}
	if _, ok := New("human", false).(*HumanLogger); !ok {
		t.Fatal("expected HumanLogger for non-json format")
	}
}

func TestHumanLogger_DebugSuppressedWhenDisabled(t *testing.T) {
	var buf bytes.Buffer
	log := &HumanLogger{out: &buf, debug: false}
	log.Debug("hidden", "k", "v")
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestHumanLogger_InfoFormatsArgs(t *testing.T) {
	var buf bytes.Buffer
	log := &HumanLogger{out: &buf, debug: true}
	log.Info("hello", "user", 7, "mode", "test")

	out := buf.String()
	if !strings.Contains(out, "INFO") {
		t.Fatalf("output missing INFO: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("output missing message: %q", out)
	}
	if !strings.Contains(out, "user=7 mode=test") {
		t.Fatalf("output missing formatted args: %q", out)
	}
}

func TestJSONLogger_DebugSuppressedWhenDisabled(t *testing.T) {
	var buf bytes.Buffer
	log := &JSONLogger{out: &buf, debug: false}
	log.Debug("hidden", "k", "v")
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestJSONLogger_OutputsStructuredEntry(t *testing.T) {
	var buf bytes.Buffer
	log := &JSONLogger{out: &buf, debug: true}
	log.Error("boom", "user", 9, "ok", true)

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
		t.Fatalf("Unmarshal: %v; raw=%q", err, buf.String())
	}
	if entry["level"] != "error" {
		t.Fatalf("level = %v, want error", entry["level"])
	}
	if entry["msg"] != "boom" {
		t.Fatalf("msg = %v, want boom", entry["msg"])
	}
	if entry["user"] != float64(9) {
		t.Fatalf("user = %v, want 9", entry["user"])
	}
	if entry["ok"] != true {
		t.Fatalf("ok = %v, want true", entry["ok"])
	}
	if _, ok := entry["time"].(string); !ok {
		t.Fatalf("time missing or not string: %v", entry["time"])
	}
}

func TestFormatArgs_OddLength(t *testing.T) {
	got := formatArgs([]any{"a", 1, "lonely"})
	if got != "a=1 lonely" {
		t.Fatalf("formatArgs = %q, want %q", got, "a=1 lonely")
	}
}
