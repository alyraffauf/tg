package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStringContents(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.md")
	if err := os.WriteFile(file, []byte("# hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "binary.bin")
	if err := os.WriteFile(binary, []byte{0xff, 0xfe, 0xfd}, 0o600); err != nil {
		t.Fatal(err)
	}
	oversize := filepath.Join(dir, "big.txt")
	f, err := os.Create(oversize)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxStringContents + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		stdin        string
		args         []string
		filenameFlag string
		wantContents string
		wantFilename string
		wantErr      bool
	}{
		{name: "file with basename", args: []string{file}, wantContents: "# hello", wantFilename: "hello.md"},
		{name: "file with flag override", args: []string{file}, filenameFlag: "custom.md", wantContents: "# hello", wantFilename: "custom.md"},
		{name: "missing file", args: []string{filepath.Join(dir, "nope")}, wantErr: true},
		{name: "empty file", args: []string{empty}, wantErr: true},
		{name: "non-UTF-8 file", args: []string{binary}, wantErr: true},
		{name: "oversize file", args: []string{oversize}, wantErr: true},
		{name: "stdin without filename", stdin: "# hello", wantErr: true},
		{name: "stdin with filename", stdin: "# from stdin", args: []string{"-"}, filenameFlag: "stdin.md", wantContents: "# from stdin", wantFilename: "stdin.md"},
		{name: "stdin with filename but empty", args: []string{"-"}, filenameFlag: "stdin.md", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents, filename, err := stringContents(strings.NewReader(tt.stdin), tt.args, tt.filenameFlag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("stringContents() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if contents != tt.wantContents {
				t.Errorf("contents = %q, want %q", contents, tt.wantContents)
			}
			if filename != tt.wantFilename {
				t.Errorf("filename = %q, want %q", filename, tt.wantFilename)
			}
		})
	}
}
