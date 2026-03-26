package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Logger is the interface for all log operations in the application.
// All implementations must handle args as key-value pairs: key1, val1, key2, val2, ...
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// New returns a Logger based on the requested format.
// format: "json" returns JSONLogger; anything else returns HumanLogger.
// debug: when false, Debug calls are suppressed.
func New(format string, debug bool) Logger {
	if format == "json" {
		return &JSONLogger{out: os.Stdout, debug: debug}
	}
	return &HumanLogger{out: os.Stdout, debug: debug}
}

// -- ANSI color codes ---------------------------------------------------------

const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// -- HumanLogger --------------------------------------------------------------

// HumanLogger writes colored, human-readable log lines to out.
type HumanLogger struct {
	out   io.Writer
	debug bool
}

func (h *HumanLogger) log(color, level, msg string, args []any) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("%s%s%s %s%-5s%s %s", colorGray, ts, colorReset, color, level, colorReset, msg)
	if len(args) > 0 {
		line += " " + formatArgs(args)
	}
	fmt.Fprintln(h.out, line)
}

func (h *HumanLogger) Debug(msg string, args ...any) {
	if !h.debug {
		return
	}
	h.log(colorGray, "DEBUG", msg, args)
}

func (h *HumanLogger) Info(msg string, args ...any) {
	h.log(colorCyan, "INFO", msg, args)
}

func (h *HumanLogger) Warn(msg string, args ...any) {
	h.log(colorYellow, "WARN", msg, args)
}

func (h *HumanLogger) Error(msg string, args ...any) {
	h.log(colorRed, "ERROR", msg, args)
}

// -- JSONLogger ---------------------------------------------------------------

// JSONLogger writes one JSON object per log line to out.
type JSONLogger struct {
	out   io.Writer
	debug bool
}

func (j *JSONLogger) log(level, msg string, args []any) {
	entry := map[string]any{
		"level": level,
		"time":  time.Now().Format(time.RFC3339),
		"msg":   msg,
	}
	for i := 0; i+1 < len(args); i += 2 {
		key := fmt.Sprintf("%v", args[i])
		entry[key] = args[i+1]
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(j.out, string(data))
}

func (j *JSONLogger) Debug(msg string, args ...any) {
	if !j.debug {
		return
	}
	j.log("debug", msg, args)
}

func (j *JSONLogger) Info(msg string, args ...any)  { j.log("info", msg, args) }
func (j *JSONLogger) Warn(msg string, args ...any)  { j.log("warn", msg, args) }
func (j *JSONLogger) Error(msg string, args ...any) { j.log("error", msg, args) }

// -- helpers ------------------------------------------------------------------

// formatArgs renders key-value pairs as "key=value key2=value2".
// Odd-length or non-string keys are rendered best-effort.
func formatArgs(args []any) string {
	out := ""
	for i := 0; i+1 < len(args); i += 2 {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%v=%v", args[i], args[i+1])
	}
	// Handle a trailing unpaired arg.
	if len(args)%2 != 0 {
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("%v", args[len(args)-1])
	}
	return out
}
