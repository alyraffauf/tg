package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

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

func TestTargetFromCWDVerifiesCustomKnotAgainstRepoRecord(t *testing.T) {
	tests := []struct {
		name       string
		candidates []gitutil.RepoContext
		recordKnot string
		want       Target
		wantErr    string
	}{
		{
			name:       "custom Knot matches record",
			candidates: []gitutil.RepoContext{{KnotHost: "KNOT.EXAMPLE", Handle: "owner.test", Repo: "example"}},
			recordKnot: "knot.example", want: Target{Handle: "owner.test", Repo: "example"},
		},
		{
			name:       "custom Knot mismatch is rejected",
			candidates: []gitutil.RepoContext{{KnotHost: "other.example", Handle: "owner.test", Repo: "example"}},
			recordKnot: "knot.example", wantErr: "does not match repository record Knot",
		},
		{
			name: "later matching candidate is accepted",
			candidates: []gitutil.RepoContext{
				{KnotHost: "wrong.example", Handle: "owner.test", Repo: "example"},
				{KnotHost: "knot.example", Handle: "owner.test", Repo: "example"},
			},
			recordKnot: "knot.example", want: Target{Handle: "owner.test", Repo: "example"},
		},
		{
			name:       "hosted endpoint needs no Knot comparison",
			candidates: []gitutil.RepoContext{{Handle: "owner.test", Repo: "example"}},
			recordKnot: "knot.example", want: Target{Handle: "owner.test", Repo: "example"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := testService(&testPDS{}, &testGit{repoCandidates: tt.candidates}, &testKnot{})
			service.appview = testAppview{repo: &tangled.Repo{
				URI:   "at://did:plc:owner/sh.tangled.repo/example",
				Value: tangledlex.Repo{Knot: tt.recordKnot},
			}}

			got, err := service.TargetFromCWD(context.Background())
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("TargetFromCWD() error = %v, want containing %q", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), "pass the repository as handle/repo") {
					t.Fatalf("TargetFromCWD() error = %v, want explicit-target hint", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("TargetFromCWD() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("TargetFromCWD() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRepoFromCWDReturnsVerifiedCustomKnotRecord(t *testing.T) {
	wantRecord := &tangled.Repo{
		URI:   "at://did:plc:owner/sh.tangled.repo/example",
		Value: tangledlex.Repo{Knot: "knot.example"},
	}
	service := testService(&testPDS{}, &testGit{repoCandidates: []gitutil.RepoContext{{
		KnotHost: "knot.example", Handle: "owner.test", Repo: "example",
	}}}, &testKnot{})
	service.appview = testAppview{repo: wantRecord}

	target, record, err := service.repoFromCWD(context.Background())
	if err != nil {
		t.Fatalf("repoFromCWD() error = %v", err)
	}
	if target != (Target{Handle: "owner.test", Repo: "example"}) {
		t.Fatalf("repoFromCWD() target = %+v", target)
	}
	if record != wantRecord {
		t.Fatalf("repoFromCWD() record = %+v, want %+v", record, wantRecord)
	}
}

func TestTargetFromCWDResolvesRepositoryPermalink(t *testing.T) {
	const repoDID = "did:plc:repository"
	gitClient := &testGit{repoCandidates: []gitutil.RepoContext{{RepoDID: repoDID}}}
	knotClient := &testKnot{description: &knot.RepoDescription{
		RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "current-rkey",
	}}
	service := testService(&testPDS{}, gitClient, knotClient)
	service.resolver = testResolver{
		identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example"),
	}
	var requestedURI string
	service.appview = testAppview{
		getRepoHook: func(uri string) { requestedURI = uri },
		repo: &tangled.Repo{Value: tangledlex.Repo{
			Name: optionalString("current-name"), Knot: "knot.example", RepoDid: optionalString(repoDID),
		}},
	}

	target, err := service.TargetFromCWD(context.Background())
	if err != nil {
		t.Fatalf("TargetFromCWD() error = %v", err)
	}
	if target != (Target{Handle: "owner.test", Repo: "current-name", RepoDID: repoDID, ownerDID: "did:plc:owner"}) {
		t.Fatalf("TargetFromCWD() = %+v", target)
	}
	if requestedURI != "at://did:plc:owner/sh.tangled.repo/current-rkey" {
		t.Fatalf("GetRepo() URI = %q", requestedURI)
	}
	factory := service.knot.(*testKnotFactory)
	if len(factory.hosts) != 1 || factory.hosts[0] != "knot.example" {
		t.Fatalf("Knot hosts = %v", factory.hosts)
	}
}

func TestTargetFromCWDResolvesRepositoryPermalinkWithCustomKnotPort(t *testing.T) {
	const repoDID = "did:plc:repository"
	knotClient := &testKnot{description: &knot.RepoDescription{
		RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo",
	}}
	service := testService(&testPDS{}, &testGit{repoCandidates: []gitutil.RepoContext{{
		RepoDID: repoDID, KnotHost: "KNOT.EXAMPLE:8443",
	}}}, knotClient)
	service.resolver = testResolver{
		identity: repositoryIdentity(repoDID, "owner.test", "https://KNOT.EXAMPLE:8443"),
	}
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Knot: "knot.example", RepoDid: optionalString(repoDID),
	}}}

	target, err := service.TargetFromCWD(context.Background())
	if err != nil {
		t.Fatalf("TargetFromCWD() error = %v", err)
	}
	if target.RepoDID != repoDID {
		t.Fatalf("TargetFromCWD() = %+v", target)
	}
	factory := service.knot.(*testKnotFactory)
	if len(factory.hosts) != 1 || factory.hosts[0] != "knot.example:8443" {
		t.Fatalf("Knot hosts = %v", factory.hosts)
	}
}

func TestResolveRepoUsesOwnerDIDWithoutResolvingHandle(t *testing.T) {
	tests := []Target{
		{Handle: "owner.test", Repo: "example", ownerDID: "did:plc:owner"},
		{Handle: "did:plc:owner", Repo: "example"},
	}
	for _, target := range tests {
		t.Run(target.String(), func(t *testing.T) {
			service := testService(&testPDS{}, &testGit{}, &testKnot{})
			service.resolver = didOnlyResolver{testResolver: testResolver{
				identity: &identity.Identity{DID: syntax.DID("did:plc:owner")},
			}}
			var requestedURI string
			service.appview = testAppview{
				getRepoHook: func(uri string) { requestedURI = uri },
				repo:        &tangled.Repo{Value: tangledlex.Repo{Knot: "knot.example"}},
			}

			if _, err := service.resolveRepo(context.Background(), target); err != nil {
				t.Fatalf("resolveRepo() error = %v", err)
			}
			if requestedURI != "at://did:plc:owner/sh.tangled.repo/example" {
				t.Fatalf("GetRepo() URI = %q", requestedURI)
			}
		})
	}
}

func TestResolveRepoUsesStableRepositoryDID(t *testing.T) {
	const repoDID = "did:plc:repository"
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	var requestedDID string
	service.appview = testAppview{
		getRepoHook: func(uri string) { t.Fatalf("unexpected name lookup %q", uri) },
		getRepoByDIDHook: func(did string) {
			requestedDID = did
		},
		repoByDID: &tangled.Repo{
			URI:   "at://did:plc:new-owner/sh.tangled.repo/new-name",
			Value: tangledlex.Repo{RepoDid: optionalString(repoDID)},
		},
	}

	repo, err := service.resolveRepo(context.Background(), Target{
		Handle: "old-owner.test", Repo: "old-name", RepoDID: repoDID, ownerDID: "did:plc:old-owner",
	})
	if err != nil {
		t.Fatalf("resolveRepo() error = %v", err)
	}
	if requestedDID != repoDID || repo.URI != "at://did:plc:new-owner/sh.tangled.repo/new-name" {
		t.Fatalf("resolveRepo() requested %q and returned %+v", requestedDID, repo)
	}
}

func TestRepositoryKnotServiceURL(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]identity.ServiceEndpoint
		want     string
		wantErr  bool
	}{
		{
			name: "current service",
			services: map[string]identity.ServiceEndpoint{
				"tangled_knot": {Type: "TangledKnot", URL: "https://current.example"},
			},
			want: "https://current.example",
		},
		{
			name: "current service preferred over legacy",
			services: map[string]identity.ServiceEndpoint{
				"tangled_knot": {Type: "TangledKnot", URL: "https://current.example"},
				"atproto_pds":  {Type: "AtprotoPersonalDataServer", URL: "https://legacy.example"},
			},
			want: "https://current.example",
		},
		{
			name: "legacy service",
			services: map[string]identity.ServiceEndpoint{
				"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://legacy.example"},
			},
			want: "https://legacy.example",
		},
		{
			name: "wrong service type",
			services: map[string]identity.ServiceEndpoint{
				"tangled_knot": {Type: "AtprotoPersonalDataServer", URL: "https://wrong.example"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repositoryKnotServiceURL(&identity.Identity{Services: tt.services})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("repositoryKnotServiceURL() = %q, want error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("repositoryKnotServiceURL() = %q, %v, want %q", got, err, tt.want)
			}
		})
	}
}

func TestTargetFromCWDRejectsUnverifiedRepositoryPermalink(t *testing.T) {
	const repoDID = "did:plc:repository"
	tests := []struct {
		name        string
		remoteKnot  string
		serviceURL  string
		description *knot.RepoDescription
		recordDID   string
		recordKnot  string
		describeErr error
		want        string
	}{
		{name: "remote Knot mismatch", remoteKnot: "other.example", serviceURL: "https://knot.example", want: "does not match repository DID Knot"},
		{name: "described DID mismatch", serviceURL: "https://knot.example", description: &knot.RepoDescription{RepoDID: "did:plc:other", OwnerDID: "did:plc:owner", RKey: "repo"}, want: "described repository DID"},
		{name: "record DID mismatch", serviceURL: "https://knot.example", description: &knot.RepoDescription{RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo"}, recordDID: "did:plc:other", recordKnot: "knot.example", want: "has repository DID"},
		{name: "record Knot mismatch", serviceURL: "https://knot.example", description: &knot.RepoDescription{RepoDID: repoDID, OwnerDID: "did:plc:owner", RKey: "repo"}, recordDID: repoDID, recordKnot: "other.example", want: "does not match repository DID Knot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			knotClient := &testKnot{description: tt.description, describeErr: tt.describeErr}
			service := testService(&testPDS{}, &testGit{repoCandidates: []gitutil.RepoContext{{
				RepoDID: repoDID, KnotHost: tt.remoteKnot,
			}}}, knotClient)
			service.resolver = testResolver{identity: repositoryIdentity(repoDID, "owner.test", tt.serviceURL)}
			service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
				Knot: tt.recordKnot, RepoDid: optionalString(tt.recordDID),
			}}}

			_, err := service.TargetFromCWD(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("TargetFromCWD() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestTargetFromCWDFallsBackToAppviewForLegacyKnot(t *testing.T) {
	const repoDID = "did:plc:repository"
	unsupportedErrors := []struct {
		name string
		err  error
	}{
		{name: "unsupported XRPC endpoint", err: &atclient.APIError{StatusCode: http.StatusNotFound, Name: "XRPCNotSupported"}},
		{name: "unimplemented method", err: &atclient.APIError{StatusCode: http.StatusNotImplemented, Name: "MethodNotImplemented"}},
	}
	for _, unsupported := range unsupportedErrors {
		t.Run(unsupported.name, func(t *testing.T) {
			service := testService(&testPDS{}, &testGit{repoCandidates: []gitutil.RepoContext{{RepoDID: repoDID}}}, &testKnot{
				describeErr: unsupported.err,
			})
			service.resolver = testResolver{identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example")}
			service.appview = testAppview{repoByDID: &tangled.Repo{
				URI: "at://did:plc:owner/sh.tangled.repo/current-rkey",
				Value: tangledlex.Repo{
					Name: optionalString("current-name"), Knot: "knot.example", RepoDid: optionalString(repoDID),
				},
			}}

			target, err := service.TargetFromCWD(context.Background())
			if err != nil {
				t.Fatalf("TargetFromCWD() error = %v", err)
			}
			if target.RepoDID != repoDID || target.Handle != "owner.test" || target.Repo != "current-name" {
				t.Fatalf("TargetFromCWD() = %+v", target)
			}
		})
	}
}

func TestResolveRepoDIDDoesNotFallBackAfterDescribeFailure(t *testing.T) {
	const repoDID = "did:plc:repository"
	tests := []struct {
		name string
		err  error
	}{
		{name: "repository not found", err: &atclient.APIError{StatusCode: http.StatusNotFound, Name: "RepoNotFound"}},
		{name: "network failure", err: errors.New("dial tcp: connection refused")},
		{name: "timeout", err: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := testService(&testPDS{}, &testGit{}, &testKnot{describeErr: test.err})
			service.resolver = testResolver{identity: repositoryIdentity(repoDID, "owner.test", "https://knot.example")}
			service.appview = testAppview{
				getRepoByDIDHook: func(string) { t.Fatal("unexpected appview fallback") },
				repoByDID: &tangled.Repo{
					URI: "at://did:plc:old-owner/sh.tangled.repo/old-name",
					Value: tangledlex.Repo{
						Knot: "knot.example", RepoDid: optionalString(repoDID),
					},
				},
			}

			_, _, err := service.resolveRepoDID(context.Background(), repoDID, "")
			if !errors.Is(err, test.err) {
				t.Fatalf("resolveRepoDID() error = %v, want wrapped %v", err, test.err)
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

func TestResolveRepoDIDRejectsInvalidKnotEndpointBeforeNetworkCall(t *testing.T) {
	const repoDID = "did:plc:repository"
	tests := []string{
		"http://knot.example",
		"https://user@knot.example",
		"https://knot.example/xrpc",
		"https://knot.example?query=yes",
		"https://knot.example/#fragment",
		"https://knot.example:0/",
		"https://knot.example:65536/",
	}
	for _, serviceURL := range tests {
		t.Run(serviceURL, func(t *testing.T) {
			service := testService(&testPDS{}, &testGit{}, &testKnot{})
			service.resolver = testResolver{
				identity: repositoryIdentity(repoDID, "owner.test", serviceURL),
			}

			_, _, err := service.resolveRepoDID(context.Background(), repoDID, "")
			if err == nil {
				t.Fatal("resolveRepoDID() error = nil")
			}
			factory := service.knot.(*testKnotFactory)
			if len(factory.hosts) != 0 {
				t.Fatalf("Knot hosts = %v, want no network client", factory.hosts)
			}
		})
	}
}

func repositoryIdentity(repoDID, ownerHandle, knotURL string) *identity.Identity {
	return &identity.Identity{
		DID:    syntax.DID(repoDID),
		Handle: syntax.Handle(ownerHandle),
		Services: map[string]identity.ServiceEndpoint{
			"tangled_knot": {Type: "TangledKnot", URL: knotURL},
		},
	}
}

func TestTargetString(t *testing.T) {
	target := Target{Handle: "aly.codes", Repo: "tg"}
	if got := target.String(); got != "aly.codes/tg" {
		t.Fatalf("String() = %q, want %q", got, "aly.codes/tg")
	}
}

type didOnlyResolver struct {
	testResolver
}

func (didOnlyResolver) ResolveHandle(context.Context, string) (*identity.Identity, error) {
	return nil, errors.New("unexpected handle resolution")
}
