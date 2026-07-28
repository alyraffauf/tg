package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRenderDetail(t *testing.T) {
	origTerm := isTerminal
	origWidth := terminalWidth
	t.Cleanup(func() {
		isTerminal = origTerm
		terminalWidth = origWidth
	})

	fields := []detailField{
		{"Title", "add config tunables"},
		{"Author", "aly.codes"},
		{"Created", "2026-07-17T05:33:20+03:00"},
	}
	const body = "likely should use viper: https://github.com/spf13/viper"

	t.Run("piped renders plain labels and verbatim body", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return false }
		var buf bytes.Buffer
		renderDetail(&buf, fields, body)
		out := buf.String()

		for _, want := range []string{"Title:", "Author:", "Created:", "add config tunables", "aly.codes"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
		if !strings.Contains(out, "\n"+body+"\n") {
			t.Errorf("piped body should be verbatim:\n%s", out)
		}
		if strings.Contains(out, "\x1b") {
			t.Errorf("piped output should have no ANSI escapes:\n%q", out)
		}
	})

	t.Run("tty renders labels and markdown body", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return true }
		terminalWidth = func(io.Writer) int { return 80 }
		var buf bytes.Buffer
		renderDetail(&buf, fields, body)
		out := buf.String()

		for _, want := range []string{"Title:", "Author:", "Created:", "add config tunables", "aly.codes"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q:\n%s", want, out)
			}
		}
		if !strings.Contains(out, "likely should use viper") {
			t.Errorf("tty body text should be present:\n%s", out)
		}
	})

	t.Run("empty body omits body section", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return false }
		var buf bytes.Buffer
		renderDetail(&buf, fields, "")
		out := buf.String()

		if strings.HasSuffix(out, "\n\n") {
			t.Errorf("empty body should not leave a trailing blank section:\n%q", out)
		}
		if strings.Contains(out, body) {
			t.Errorf("empty body should not emit body text:\n%s", out)
		}
	})

	t.Run("labels align to widest", func(t *testing.T) {
		isTerminal = func(io.Writer) bool { return false }
		var buf bytes.Buffer
		unevenFields := []detailField{
			{"A", "va"},
			{"LongerLabel", "vb"},
		}
		renderDetail(&buf, unevenFields, "")
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines, got %d", len(lines))
		}
		colA := strings.Index(lines[0], "va")
		colB := strings.Index(lines[1], "vb")
		if colA < 0 || colB < 0 {
			t.Fatalf("values not found:\n%s", buf.String())
		}
		if colA != colB {
			t.Errorf("values not aligned: col %d vs %d\n%s", colA, colB, buf.String())
		}
	})

	t.Run("label column width accounts for colon and separator", func(t *testing.T) {
		got := labelColumnWidth([]detailField{{"Title", ""}, {"Author", ""}})
		const want = len("Author") + 2
		if got != want {
			t.Errorf("labelColumnWidth = %d, want %d", got, want)
		}
	})
}
