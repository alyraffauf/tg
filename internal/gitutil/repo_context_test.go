package gitutil

import (
	"slices"
	"testing"

	"github.com/alyraffauf/tg/knot"
)

func TestParseRepoCandidate(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantOK     bool
		wantKnot   string
		wantHandle string
		wantRepo   string
	}{
		{"ssh scp-like", "git@tangled.org:aly.codes/tg", true, "", "aly.codes", "tg"},
		{"ssh scp-like with .git", "git@tangled.org:aly.codes/tg.git", true, "", "aly.codes", "tg"},
		{"ssh scp-like no user", "tangled.org:aly.codes/tg", true, "", "aly.codes", "tg"},
		{"ssh scp-like trailing slash", "git@tangled.org:aly.codes/tg/", true, "", "aly.codes", "tg"},
		{"ssh:// with user", "ssh://git@tangled.org/aly.codes/tg", true, "", "aly.codes", "tg"},
		{"ssh:// without user", "ssh://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg"},
		{"ssh:// with port", "ssh://git@tangled.org:2222/aly.codes/tg", true, "", "aly.codes", "tg"},
		{"git://", "git://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg"},
		{"git:// with .git", "git://tangled.org/aly.codes/tg.git", true, "", "aly.codes", "tg"},
		{"https", "https://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg"},
		{"https with .git", "https://tangled.org/aly.codes/tg.git", true, "", "aly.codes", "tg"},
		{"https trailing slash", "https://tangled.org/aly.codes/tg/", true, "", "aly.codes", "tg"},
		{"https .git trailing slash", "https://tangled.org/aly.codes/tg.git/", true, "", "aly.codes", "tg"},
		{"https extra segment", "https://tangled.org/aly.codes/tg/extra", false, "", "", ""},
		{"http", "http://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg"},
		{"hostname case insensitive", "git@Tangled.ORG:aly.codes/tg", true, "", "aly.codes", "tg"},
		{"custom Knot ssh", "git@knot.example:aly.codes/tg", true, "knot.example", "aly.codes", "tg"},
		{"custom Knot ssh URL with port", "ssh://git@KNOT.EXAMPLE:2222/aly.codes/tg", true, "knot.example", "aly.codes", "tg"},
		{"github ssh is an untrusted candidate", "git@github.com:alyraffauf/tg.git", true, "github.com", "alyraffauf", "tg"},
		{"github https is an untrusted candidate", "https://github.com/alyraffauf/tg.git", true, "github.com", "alyraffauf", "tg"},
		{"unrelated HTTPS is an untrusted candidate", "https://example.com/foo/bar", true, "example.com", "foo", "bar"},
		{"unrelated git protocol", "git://example.com/foo/bar", false, "", "", ""},
		{"empty", "", false, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, ok := parseRepoCandidate(tt.url)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if rc.KnotHost != tt.wantKnot {
				t.Errorf("KnotHost = %q, want %q", rc.KnotHost, tt.wantKnot)
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

func TestKnotRemoteURL(t *testing.T) {
	tests := []struct {
		name     string
		knotHost string
		sshPort  int
		want     string
	}{
		{name: "hosted proxy", sshPort: 22, want: "git@tangled.org:aly.codes/tg"},
		{name: "hosted proxy and custom port", sshPort: 2222, want: "ssh://git@tangled.org:2222/aly.codes/tg"},
		{name: "default knot through hosted proxy", knotHost: knot.DefaultKnot, sshPort: 22, want: "git@tangled.org:aly.codes/tg"},
		{name: "default knot through hosted proxy and custom port", knotHost: knot.DefaultKnot, sshPort: 2222, want: "ssh://git@tangled.org:2222/aly.codes/tg"},
		{name: "custom knot and port", knotHost: "knot.secluded.site", sshPort: 2222, want: "ssh://git@knot.secluded.site:2222/aly.codes/tg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := knotRemoteURL(tt.knotHost, tt.sshPort, "aly.codes", "tg"); got != tt.want {
				t.Fatalf("knotRemoteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloneRemoteURL(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		knotHost    string
		sshPort     int
		expectedURL string
		expectedErr string
	}{
		{name: "SSH", protocol: "ssh", knotHost: "knot.example", sshPort: 22, expectedURL: "git@knot.example:aly.codes/tg"},
		{name: "HTTPS", protocol: "https", knotHost: "knot.example", expectedURL: "https://knot.example/aly.codes/tg.git"},
		{name: "HTTPS without knot", protocol: "https", expectedErr: "HTTPS clone requires a Knot host"},
		{name: "unsupported protocol", protocol: "git", expectedErr: "unsupported clone protocol \"git\""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cloneURL, err := cloneRemoteURL(testCase.protocol, testCase.knotHost, testCase.sshPort, "aly.codes", "tg")
			if testCase.expectedErr != "" {
				if err == nil || err.Error() != testCase.expectedErr {
					t.Fatalf("cloneRemoteURL() error = %v, want %q", err, testCase.expectedErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("cloneRemoteURL() error = %v", err)
			}
			if cloneURL != testCase.expectedURL {
				t.Fatalf("cloneRemoteURL() = %q, want %q", cloneURL, testCase.expectedURL)
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
