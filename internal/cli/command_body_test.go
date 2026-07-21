package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandBody(t *testing.T) {
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte("body from file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		body     string
		bodyFile string
		want     string
		wantErr  bool
	}{
		{name: "body flag", body: "inline body", want: "inline body"},
		{name: "neither", want: ""},
		{name: "body file", bodyFile: bodyFile, want: "body from file\n"},
		{name: "both", body: "inline", bodyFile: bodyFile, wantErr: true},
		{name: "missing file", bodyFile: filepath.Join(t.TempDir(), "nope.md"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := commandBody(tt.body, tt.bodyFile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
