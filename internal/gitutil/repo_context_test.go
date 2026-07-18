package gitutil

import (
	"slices"
	"testing"
)

func TestParseTangledURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantOK     bool
		wantHandle string
		wantRepo   string
	}{
		{"ssh scp-like", "git@tangled.org:aly.codes/tg", true, "aly.codes", "tg"},
		{"ssh scp-like with .git", "git@tangled.org:aly.codes/tg.git", true, "aly.codes", "tg"},
		{"ssh scp-like no user", "tangled.org:aly.codes/tg", true, "aly.codes", "tg"},
		{"ssh scp-like trailing slash", "git@tangled.org:aly.codes/tg/", true, "aly.codes", "tg"},
		{"ssh:// with user", "ssh://git@tangled.org/aly.codes/tg", true, "aly.codes", "tg"},
		{"ssh:// without user", "ssh://tangled.org/aly.codes/tg", true, "aly.codes", "tg"},
		{"ssh:// with port", "ssh://git@tangled.org:2222/aly.codes/tg", true, "aly.codes", "tg"},
		{"git://", "git://tangled.org/aly.codes/tg", true, "aly.codes", "tg"},
		{"git:// with .git", "git://tangled.org/aly.codes/tg.git", true, "aly.codes", "tg"},
		{"https", "https://tangled.org/aly.codes/tg", true, "aly.codes", "tg"},
		{"https with .git", "https://tangled.org/aly.codes/tg.git", true, "aly.codes", "tg"},
		{"https trailing slash", "https://tangled.org/aly.codes/tg/", true, "aly.codes", "tg"},
		{"https .git trailing slash", "https://tangled.org/aly.codes/tg.git/", true, "aly.codes", "tg"},
		{"https extra segment", "https://tangled.org/aly.codes/tg/extra", false, "", ""},
		{"http", "http://tangled.org/aly.codes/tg", true, "aly.codes", "tg"},
		{"hostname case insensitive", "git@Tangled.ORG:aly.codes/tg", true, "aly.codes", "tg"},
		{"github ssh", "git@github.com:alyraffauf/tg.git", false, "", ""},
		{"github https", "https://github.com/alyraffauf/tg.git", false, "", ""},
		{"unrelated", "https://example.com/foo/bar", false, "", ""},
		{"empty", "", false, "", ""},
		{"ssh wrong host", "git@example.org:aly.codes/tg", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, ok := parseTangledURL(tt.url)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if rc.Handle != tt.wantHandle {
				t.Errorf("Handle = %q, want %q", rc.Handle, tt.wantHandle)
			}
			if rc.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", rc.Repo, tt.wantRepo)
			}
		})
	}
}

func TestOriginFirst(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"origin first", []string{"upstream", "origin", "fork"}, []string{"origin", "upstream", "fork"}},
		{"no origin", []string{"upstream", "fork"}, []string{"upstream", "fork"}},
		{"only origin", []string{"origin"}, []string{"origin"}},
		{"empty", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := originFirst(tt.input); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
