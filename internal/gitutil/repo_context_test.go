package gitutil

import (
	"slices"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/knot"
)

func TestParseRepoCandidate(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOK      bool
		wantKnot    string
		wantHandle  string
		wantRepo    string
		wantRepoDID string
	}{
		{"ssh scp-like", "git@tangled.org:aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"ssh scp-like with .git", "git@tangled.org:aly.codes/tg.git", true, "", "aly.codes", "tg", ""},
		{"ssh scp-like no user", "tangled.org:aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"ssh scp-like trailing slash", "git@tangled.org:aly.codes/tg/", true, "", "aly.codes", "tg", ""},
		{"ssh:// with user", "ssh://git@tangled.org/aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"ssh:// without user", "ssh://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"ssh:// with port", "ssh://git@tangled.org:2222/aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"git://", "git://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"git:// with .git", "git://tangled.org/aly.codes/tg.git", true, "", "aly.codes", "tg", ""},
		{"https", "https://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"https with .git", "https://tangled.org/aly.codes/tg.git", true, "", "aly.codes", "tg", ""},
		{"https trailing slash", "https://tangled.org/aly.codes/tg/", true, "", "aly.codes", "tg", ""},
		{"https .git trailing slash", "https://tangled.org/aly.codes/tg.git/", true, "", "aly.codes", "tg", ""},
		{"https extra segment", "https://tangled.org/aly.codes/tg/extra", false, "", "", "", ""},
		{"http", "http://tangled.org/aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"hostname case insensitive", "git@Tangled.ORG:aly.codes/tg", true, "", "aly.codes", "tg", ""},
		{"SSH permalink", "git@tangled.org:did:plc:repo", true, "", "", "", "did:plc:repo"},
		{"HTTPS permalink", "https://tangled.org/did:plc:repo", true, "", "", "", "did:plc:repo"},
		{"DID ending in .git", "https://tangled.org/did:web:forge.git/", true, "", "", "", "did:web:forge.git"},
		{"custom Knot permalink", "ssh://git@knot.example/did:plc:repo", true, "knot.example", "", "", "did:plc:repo"},
		{"invalid one-segment path", "git@tangled.org:not-a-did", false, "", "", "", ""},
		{"custom Knot ssh", "git@knot.example:aly.codes/tg", true, "knot.example", "aly.codes", "tg", ""},
		{"custom Knot ssh URL with port", "ssh://git@KNOT.EXAMPLE:2222/aly.codes/tg", true, "knot.example", "aly.codes", "tg", ""},
		{"github ssh is an untrusted candidate", "git@github.com:alyraffauf/tg.git", true, "github.com", "alyraffauf", "tg", ""},
		{"github https is an untrusted candidate", "https://github.com/alyraffauf/tg.git", true, "github.com", "alyraffauf", "tg", ""},
		{"unrelated HTTPS is an untrusted candidate", "https://example.com/foo/bar", true, "example.com", "foo", "bar", ""},
		{"unrelated git protocol", "git://example.com/foo/bar", false, "", "", "", ""},
		{"empty", "", false, "", "", "", ""},
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
			if rc.RepoDID != tt.wantRepoDID {
				t.Errorf("RepoDID = %q, want %q", rc.RepoDID, tt.wantRepoDID)
			}
		})
	}
}

func TestKnotRemoteURL(t *testing.T) {
	tests := []struct {
		name     string
		knotHost string
		sshPort  int
		repoPath string
		want     string
	}{
		{name: "hosted proxy", sshPort: 22, want: "git@tangled.org:aly.codes/tg"},
		{name: "hosted proxy and custom port", sshPort: 2222, want: "ssh://git@tangled.org:2222/aly.codes/tg"},
		{name: "default knot through hosted proxy", knotHost: knot.DefaultKnot, sshPort: 22, want: "git@tangled.org:aly.codes/tg"},
		{name: "default knot through hosted proxy and custom port", knotHost: knot.DefaultKnot, sshPort: 2222, want: "ssh://git@tangled.org:2222/aly.codes/tg"},
		{name: "custom knot and port", knotHost: "knot.secluded.site", sshPort: 2222, want: "ssh://git@knot.secluded.site:2222/aly.codes/tg"},
		{name: "repository DID", knotHost: "knot.example", sshPort: 22, want: "git@knot.example:did:plc:repo", repoPath: "did:plc:repo"},
		{name: "repository DID and custom port", knotHost: "knot.example", sshPort: 2222, want: "ssh://git@knot.example:2222/did:plc:repo", repoPath: "did:plc:repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := tt.repoPath
			if repoPath == "" {
				repoPath = "aly.codes/tg"
			}
			if got := knotRemoteURL(tt.knotHost, tt.sshPort, repoPath); got != tt.want {
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
		repoDID     string
		expectedURL string
		expectedErr string
	}{
		{name: "SSH", protocol: "ssh", knotHost: "knot.example", sshPort: 22, expectedURL: "git@knot.example:aly.codes/tg"},
		{name: "SSH permalink", protocol: "ssh", knotHost: "knot.example", sshPort: 22, repoDID: "did:plc:repo", expectedURL: "git@knot.example:did:plc:repo"},
		{name: "SSH permalink with custom port", protocol: "ssh", knotHost: "knot.example", sshPort: 2222, repoDID: "did:plc:repo", expectedURL: "ssh://git@knot.example:2222/did:plc:repo"},
		{name: "SSH permalink without Knot", protocol: "ssh", sshPort: 22, repoDID: "did:plc:repo", expectedURL: "git@tangled.org:did:plc:repo"},
		{name: "HTTPS", protocol: "https", knotHost: "knot.example", expectedURL: "https://knot.example/aly.codes/tg.git"},
		{name: "HTTPS permalink", protocol: "https", knotHost: "knot.example", repoDID: "did:plc:repo", expectedURL: "https://knot.example/did:plc:repo"},
		{name: "HTTPS permalink without Knot", protocol: "https", repoDID: "did:plc:repo", expectedURL: "https://tangled.org/did:plc:repo"},
		{name: "invalid permalink", protocol: "ssh", repoDID: "not-a-did", expectedErr: "invalid repository DID \"not-a-did\""},
		{name: "HTTPS without knot", protocol: "https", expectedErr: "HTTPS clone requires a Knot host"},
		{name: "unsupported protocol", protocol: "git", expectedErr: "unsupported clone protocol \"git\""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cloneURL, err := cloneRemoteURL(CloneRepoParams{
				Protocol: testCase.protocol, KnotHost: testCase.knotHost, SSHPort: testCase.sshPort,
				Handle: "aly.codes", Repo: "tg", RepoDID: testCase.repoDID,
			})
			if testCase.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.expectedErr) {
					t.Fatalf("cloneRemoteURL() error = %v, want containing %q", err, testCase.expectedErr)
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
