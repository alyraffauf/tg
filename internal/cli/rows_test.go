package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestShortDate(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		want      string
	}{
		{name: "iso timestamp", timestamp: "2026-07-21T10:30:00Z", want: "2026-07-21"},
		{name: "date only", timestamp: "2026-07-21", want: "2026-07-21"},
		{name: "short string", timestamp: "2026", want: "2026"},
		{name: "empty", timestamp: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortDate(tt.timestamp); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTable(t *testing.T) {
	origTerm := isTerminal
	origWidth := terminalWidth
	t.Cleanup(func() {
		isTerminal = origTerm
		terminalWidth = origWidth
	})

	header := []string{"RKEY", "TITLE"}
	rows := [][]string{
		{"1", "Fix bug"},
		{"2", "Add feature"},
	}

	t.Run("empty prints message", func(t *testing.T) {
		var buf bytes.Buffer
		renderTable(&buf, header, nil, "nothing here")
		if got := buf.String(); got != "nothing here\n" {
			t.Fatalf("got %q, want %q", got, "nothing here\n")
		}
	})

	t.Run("piped is plain", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return false }
		var buf bytes.Buffer
		renderTable(&buf, header, rows, "")
		out := buf.String()

		// One line per header + row.
		if got := strings.Count(out, "\n"); got != len(rows)+1 {
			t.Fatalf("got %d lines, want %d", got, len(rows)+1)
		}
		for _, want := range []string{"RKEY", "TITLE", "1", "Fix bug", "2", "Add feature"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
		if strings.ContainsAny(out, "│─╭╮╰╯├┤┬┴┼") {
			t.Errorf("piped output should have no border glyphs:\n%s", out)
		}
	})

	t.Run("tty is bordered", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return true }
		var buf bytes.Buffer
		renderTable(&buf, header, rows, "")
		out := buf.String()

		for _, want := range []string{"RKEY", "TITLE", "1", "Fix bug", "2", "Add feature"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
		if !strings.ContainsAny(out, "│─") {
			t.Errorf("tty output should have border glyphs:\n%s", out)
		}
	})

	t.Run("tty with unknown width renders natural", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return true }
		terminalWidth = func(io.Writer) int { return 0 }

		longRows := [][]string{
			{"3jz7x", "Fix leak in the resolver cache layer"},
		}
		var buf bytes.Buffer
		renderTable(&buf, []string{"RKEY", "TITLE"}, longRows, "")
		out := buf.String()

		if !strings.Contains(out, "Fix leak in the resolver cache layer") {
			t.Errorf("natural render should not truncate:\n%s", out)
		}
		if strings.Contains(out, "…") {
			t.Errorf("natural render should not truncate:\n%s", out)
		}
		if !strings.ContainsAny(out, "│─") {
			t.Errorf("tty output should have border glyphs:\n%s", out)
		}
	})

	t.Run("narrow terminal shrinks to fit", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return true }
		const cols = 40
		terminalWidth = func(io.Writer) int { return cols }

		narrowRows := [][]string{
			{"3jz7x", "Fix leak in the resolver cache layer"},
			{"3k2ab", "Support --json output for repo list command"},
		}
		var buf bytes.Buffer
		renderTable(&buf, []string{"RKEY", "TITLE"}, narrowRows, "")
		out := buf.String()

		if !strings.Contains(out, "…") {
			t.Errorf("narrow output should truncate with …:\n%s", out)
		}
		for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if w := lipgloss.Width(line); w > cols {
				t.Errorf("line %d is %d cols wide (max %d):\n%s", i, w, cols, line)
			}
		}
	})
}
