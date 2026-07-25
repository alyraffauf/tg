package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
)

func TestCreateRepoValidatesConnectionSettingsBeforeProvisioning(t *testing.T) {
	tests := []struct {
		name        string
		knotHost    string
		sshPort     int
		pushPath    string
		clone       bool
		wantHost    string
		wantErr     string
		wantClones  int
		wantCreates int
		wantPuts    int
	}{
		{name: "malformed hostname", knotHost: "https://knot.example/path", wantErr: "invalid Knot hostname"},
		{name: "zero push port", knotHost: "knot.example", pushPath: ".", wantErr: "SSH port"},
		{name: "negative push port", knotHost: "knot.example", sshPort: -1, pushPath: ".", wantErr: "SSH port"},
		{name: "push port above maximum", knotHost: "knot.example", sshPort: 65536, pushPath: ".", wantErr: "SSH port"},
		{name: "zero clone port", knotHost: "knot.example", clone: true, wantErr: "SSH port"},
		{name: "valid custom port", knotHost: "knot.example", sshPort: 2222, pushPath: ".", wantHost: "knot.example", wantCreates: 1, wantPuts: 1},
		{name: "clone from custom knot and port", knotHost: "knot.example", sshPort: 2222, clone: true, wantHost: "knot.example", wantClones: 1, wantCreates: 1, wantPuts: 1},
		{name: "hostname is canonicalized", knotHost: "KNOT.EXAMPLE", wantHost: "knot.example", wantCreates: 1, wantPuts: 1},
		{name: "unused invalid port", knotHost: "knot.example", wantHost: "knot.example", wantCreates: 1, wantPuts: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pds := &testPDS{}
			knotClient := &testKnot{}
			git := &testGit{}
			service := testService(pds, git, knotClient)

			result, err := service.CreateRepo(context.Background(), CreateRepoInput{
				KnotHost: tt.knotHost, SSHPort: tt.sshPort, Name: "example", Clone: tt.clone, PushPath: tt.pushPath,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("CreateRepo() error = %v, want containing %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("CreateRepo() error = %v", err)
				}
				if result.Knot != tt.wantHost {
					t.Fatalf("CreateRepo() Knot = %q, want %q", result.Knot, tt.wantHost)
				}
			}
			if knotClient.createCalls != tt.wantCreates || len(pds.puts) != tt.wantPuts {
				t.Fatalf("create/put calls = %d/%d, want %d/%d", knotClient.createCalls, len(pds.puts), tt.wantCreates, tt.wantPuts)
			}
			if len(git.clones) != tt.wantClones {
				t.Fatalf("clone calls = %+v, want %d", git.clones, tt.wantClones)
			}
			if tt.wantClones == 1 && (git.clones[0].KnotHost != tt.wantHost || git.clones[0].SSHPort != tt.sshPort) {
				t.Fatalf("clone destination = %+v, want %s:%d", git.clones[0], tt.wantHost, tt.sshPort)
			}
			if tt.wantErr != "" && (pds.serviceAuthCalls != 0 || len(git.pushes) != 0) {
				t.Fatalf("service auth/push calls = %d/%d, want no side effects", pds.serviceAuthCalls, len(git.pushes))
			}
			if tt.wantErr == "" {
				wantAudience := "did:web:" + tt.wantHost
				if len(pds.serviceAuthAudiences) == 0 {
					t.Fatal("service auth was not requested")
				}
				for _, audience := range pds.serviceAuthAudiences {
					if audience != wantAudience {
						t.Fatalf("service auth audience = %q, want %q", audience, wantAudience)
					}
				}
			}
		})
	}
}

func TestCreateRepoSelectsKnot(t *testing.T) {
	listFailure := errors.New("PDS unavailable")
	tests := []struct {
		name           string
		configuredKnot string
		records        []atproto.RecordItem
		listErr        error
		verifyErrors   map[string]error
		wantKnot       string
		wantErr        string
		wantWarning    string
		wantVerifies   int
		wantListCalls  int
		wantCreates    int
		clone          bool
		push           bool
	}{
		{
			name: "configured Knot bypasses discovery", configuredKnot: "configured.example",
			listErr: listFailure, wantKnot: "configured.example", wantCreates: 1,
		},
		{
			name: "one verified Knot", records: []atproto.RecordItem{
				validKnotRegistration("verified.example"),
			}, wantKnot: "verified.example", wantVerifies: 1, wantListCalls: 1, wantCreates: 1, push: true,
		},
		{
			name: "verified Knot is canonicalized", records: []atproto.RecordItem{
				validKnotRegistration("VERIFIED.EXAMPLE"),
			}, wantKnot: "verified.example", wantVerifies: 1, wantListCalls: 1, wantCreates: 1, push: true,
		},
		{
			name: "automatically selected Knot is used for clone", records: []atproto.RecordItem{
				validKnotRegistration("verified.example"),
			}, wantKnot: "verified.example", wantVerifies: 1, wantListCalls: 1, wantCreates: 1, clone: true,
		},
		{name: "no verified Knots", wantKnot: DefaultKnot, wantListCalls: 1, wantCreates: 1},
		{
			name: "multiple verified Knots", records: []atproto.RecordItem{
				validKnotRegistration("two.example"),
				validKnotRegistration("one.example"),
			}, wantErr: "multiple verified Knots found (one.example, two.example); select one with --knot or set it in the config file", wantVerifies: 2, wantListCalls: 1,
		},
		{
			name: "verified Knot is selected with warning for stale registration", records: []atproto.RecordItem{
				validKnotRegistration("verified.example"),
				validKnotRegistration("stale.example"),
			}, verifyErrors: map[string]error{"stale.example": errors.New("owner mismatch")},
			wantKnot: "verified.example", wantWarning: "stale.example: owner mismatch", wantVerifies: 2, wantListCalls: 1, wantCreates: 1,
		},
		{
			name: "unverified registration does not fall back", records: []atproto.RecordItem{
				validKnotRegistration("stale.example"),
			}, verifyErrors: map[string]error{"stale.example": errors.New("owner mismatch")},
			wantErr: "no Knot registrations could be verified", wantVerifies: 1, wantListCalls: 1,
		},
		{
			name: "duplicate normalized registration is ignored", records: []atproto.RecordItem{
				validKnotRegistration("verified.example"),
				validKnotRegistration("VERIFIED.EXAMPLE"),
			}, wantKnot: "verified.example", wantWarning: "duplicate Knot registration", wantVerifies: 1, wantListCalls: 1, wantCreates: 1,
		},
		{
			name: "malformed record URI", records: []atproto.RecordItem{{URI: "not-an-at-uri", Value: validKnotRegistrationValue()}},
			wantErr: "ignored invalid Knot registration URI", wantListCalls: 1,
		},
		{
			name: "malformed record value", records: []atproto.RecordItem{
				{URI: "at://did:plc:owner/sh.tangled.knot/invalid.example", Value: map[string]any{"$type": knotCollection}},
			}, wantErr: "invalid createdAt", wantListCalls: 1,
		},
		{
			name: "verified Knot is selected with warning for malformed record", records: []atproto.RecordItem{
				validKnotRegistration("verified.example"),
				{URI: "at://did:plc:owner/sh.tangled.knot/invalid.example", Value: map[string]any{"$type": "wrong.type", "createdAt": "2026-01-01T00:00:00Z"}},
			}, wantKnot: "verified.example", wantWarning: "$type must be", wantVerifies: 1, wantListCalls: 1, wantCreates: 1,
		},
		{
			name: "record URI with port", records: []atproto.RecordItem{
				{URI: "at://did:plc:owner/sh.tangled.knot/host:2222", Value: validKnotRegistrationValue()},
			}, wantErr: "invalid Knot hostname", wantListCalls: 1,
		},
		{
			name: "record URI with invalid DNS hostname", records: []atproto.RecordItem{
				{URI: "at://did:plc:owner/sh.tangled.knot/...", Value: validKnotRegistrationValue()},
			}, wantErr: "invalid Knot hostname", wantListCalls: 1,
		},
		{name: "list error", listErr: listFailure, wantErr: listFailure.Error(), wantListCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pds := &testPDS{records: tt.records, listErr: tt.listErr}
			knotClient := &testKnot{}
			git := &testGit{}
			service := testService(pds, git, knotClient)
			verifier := &testKnotOwnershipVerifier{errors: tt.verifyErrors}
			service.knotOwnershipVerifier = verifier
			knotFactory := &testKnotFactory{client: knotClient}
			service.knot = knotFactory
			pushPath := ""
			if tt.push {
				pushPath = "."
			}

			result, err := service.CreateRepo(context.Background(), CreateRepoInput{
				KnotHost: tt.configuredKnot, SSHPort: 22, Name: "example", Clone: tt.clone, PushPath: pushPath,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("CreateRepo() error = %v, want containing %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("CreateRepo() error = %v", err)
				}
				if result.Knot != tt.wantKnot {
					t.Fatalf("CreateRepo() Knot = %q, want %q", result.Knot, tt.wantKnot)
				}
				if tt.wantWarning != "" && (len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "; "), tt.wantWarning)) {
					t.Fatalf("CreateRepo() warnings = %v, want containing %q", result.Warnings, tt.wantWarning)
				}
			}
			if pds.listCalls != tt.wantListCalls {
				t.Fatalf("list calls = %d, want %d", pds.listCalls, tt.wantListCalls)
			}
			if knotClient.createCalls != tt.wantCreates {
				t.Fatalf("provision calls = %d, want %d", knotClient.createCalls, tt.wantCreates)
			}
			if len(verifier.hosts) != tt.wantVerifies {
				t.Fatalf("verification calls = %v, want %d", verifier.hosts, tt.wantVerifies)
			}
			if tt.wantErr != "" && (pds.serviceAuthCalls != 0 || len(pds.puts) != 0) {
				t.Fatalf("service auth/put calls = %d/%d, want no mutation side effects", pds.serviceAuthCalls, len(pds.puts))
			}
			if tt.push && (len(git.pushes) != 1 || git.pushes[0].KnotHost != tt.wantKnot) {
				t.Fatalf("pushes = %+v, want one push to %q", git.pushes, tt.wantKnot)
			}
			if tt.clone && (len(git.clones) != 1 || git.clones[0].KnotHost != tt.wantKnot) {
				t.Fatalf("clones = %+v, want one clone from %q", git.clones, tt.wantKnot)
			}
			if tt.wantErr == "" {
				if len(knotFactory.hosts) == 0 || knotFactory.hosts[0] != tt.wantKnot {
					t.Fatalf("Knot client hosts = %v, want %q", knotFactory.hosts, tt.wantKnot)
				}
				if len(pds.serviceAuthAudiences) == 0 || pds.serviceAuthAudiences[0] != "did:web:"+tt.wantKnot {
					t.Fatalf("service auth audiences = %v, want did:web:%s", pds.serviceAuthAudiences, tt.wantKnot)
				}
			}
		})
	}
}

func TestCreateRepoBoundsAutomaticKnotVerification(t *testing.T) {
	pds := &testPDS{records: make([]atproto.RecordItem, maxKnotRegistrations+1)}
	service := testService(pds, &testGit{}, &testKnot{})
	verifier := &testKnotOwnershipVerifier{}
	service.knotOwnershipVerifier = verifier

	_, err := service.CreateRepo(context.Background(), CreateRepoInput{Name: "example"})
	if err == nil || !strings.Contains(err.Error(), "found more than 10 Knot registrations") {
		t.Fatalf("CreateRepo() error = %v, want bounded-verification error", err)
	}
	if len(pds.listOptions) != 1 || pds.listOptions[0].Limit != maxKnotRegistrations+1 {
		t.Fatalf("list options = %+v, want one bounded request", pds.listOptions)
	}
	if len(verifier.hosts) != 0 || pds.serviceAuthCalls != 0 || len(pds.puts) != 0 {
		t.Fatalf("verification/auth/put side effects = %d/%d/%d, want none", len(verifier.hosts), pds.serviceAuthCalls, len(pds.puts))
	}
}

func validKnotRegistration(host string) atproto.RecordItem {
	return atproto.RecordItem{
		URI:   "at://did:plc:owner/sh.tangled.knot/" + host,
		Value: validKnotRegistrationValue(),
	}
}

func validKnotRegistrationValue() map[string]any {
	return map[string]any{
		"$type":     knotCollection,
		"createdAt": "2026-01-01T00:00:00Z",
	}
}
