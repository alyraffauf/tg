package app

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
)

func TestRepositoryLegacyKnotDiscoveryPreservesEndpointChecks(t *testing.T) {
	for _, test := range []struct {
		name       string
		pds        string
		git        *identity.ServiceEndpoint
		wantGitURL string
	}{
		{name: "legacy", pds: "https://knot.example", wantGitURL: "https://knot.example/did:plc:repo"},
		{name: "legacy port", pds: "https://knot.example:8443", wantGitURL: "https://knot.example:8443/did:plc:repo"},
		{name: "modern", pds: "https://knot.example", git: &identity.ServiceEndpoint{Type: "TangledKnot", URL: "https://knot.example/repo/tag"}, wantGitURL: "https://knot.example/repo/tag"},
		{name: "mismatched host", pds: "https://knot.example", git: &identity.ServiceEndpoint{Type: "TangledKnot", URL: "https://other.example/repo/tag"}},
		{name: "mismatched port", pds: "https://knot.example:8443", git: &identity.ServiceEndpoint{Type: "TangledKnot", URL: "https://knot.example/repo/tag"}},
		{name: "wrong type", pds: "https://knot.example", git: &identity.ServiceEndpoint{Type: "Other", URL: "https://knot.example"}},
		{name: "empty endpoint", pds: "https://knot.example", git: &identity.ServiceEndpoint{Type: "TangledKnot"}},
		{name: "legacy insecure endpoint", pds: "http://knot.example"},
		{name: "legacy mismatched bobbin", pds: "https://other.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _, _ := migrationService(t)
			ident := repositoryIdentity("did:plc:repo", "owner.test", test.pds)
			delete(ident.Services, "tangled_knot")
			if test.git != nil {
				ident.Services["tangled_knot"] = *test.git
			}
			service.resolver = testResolver{identity: ident}
			_, repo, err := service.resolveRepoDID(context.Background(), "did:plc:repo", "")
			if test.wantGitURL == "" {
				if err == nil {
					t.Fatalf("accepted invalid identity: %+v", repo)
				}
			} else if err != nil || repo.GitURL != test.wantGitURL {
				t.Fatalf("repo=%+v err=%v", repo, err)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantHandle string
		wantRepo   string
		wantErr    bool
	}{
		{name: "handle and repo", arg: "aly.codes/tg", wantHandle: "aly.codes", wantRepo: "tg"},
		{name: "did handle", arg: "did:plc:abc123/tg", wantHandle: "did:plc:abc123", wantRepo: "tg"},
		// Repo names become atproto record keys, which cannot contain "/".
		{name: "repo containing slash", arg: "aly.codes/a/b", wantErr: true},
		{name: "no slash", arg: "tg", wantErr: true},
		{name: "empty handle", arg: "/tg", wantErr: true},
		{name: "empty repo", arg: "aly.codes/", wantErr: true},
		{name: "empty", arg: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := ParseTarget(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if target.Handle != tt.wantHandle || target.Repo != tt.wantRepo {
				t.Fatalf("got (%q, %q), want (%q, %q)", target.Handle, target.Repo, tt.wantHandle, tt.wantRepo)
			}
		})
	}
}

func TestParseKnotServiceEndpoint(t *testing.T) {
	tests := []struct {
		raw           string
		wantAuthority string
		wantHostname  string
	}{
		{raw: "https://KNOT.EXAMPLE", wantAuthority: "knot.example", wantHostname: "knot.example"},
		{raw: "https://KNOT.EXAMPLE/", wantAuthority: "knot.example", wantHostname: "knot.example"},
		{raw: "https://KNOT.EXAMPLE:443/", wantAuthority: "knot.example", wantHostname: "knot.example"},
		{raw: "https://KNOT.EXAMPLE:00443/", wantAuthority: "knot.example", wantHostname: "knot.example"},
		{raw: "https://knot.example:8443/", wantAuthority: "knot.example:8443", wantHostname: "knot.example"},
		{raw: "https://knot.example/xrpc"},
		{raw: "http://knot.example"},
		{raw: "https://user@knot.example"},
		{raw: "https://knot.example?query=yes"},
		{raw: "https://knot.example/#fragment"},
		{raw: "https://knot.example:0/"},
		{raw: "https://knot.example:65536/"},
		{raw: "https://knot.example:not-a-port/"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := parseKnotServiceEndpoint(tt.raw)
			if tt.wantAuthority == "" {
				if err == nil {
					t.Fatalf("parseKnotServiceEndpoint() = %+v, want error", got)
				}
				return
			}
			want := knotServiceEndpoint{Authority: tt.wantAuthority, Hostname: tt.wantHostname}
			if err != nil || got != want {
				t.Fatalf("parseKnotServiceEndpoint() = %+v, %v, want %+v", got, err, want)
			}
		})
	}
}

func TestTargetString(t *testing.T) {
	target := Target{Handle: "aly.codes", Repo: "tg"}
	if got := target.String(); got != "aly.codes/tg" {
		t.Fatalf("String() = %q, want %q", got, "aly.codes/tg")
	}
}
