package app

import (
	"context"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
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

func TestTargetString(t *testing.T) {
	target := Target{Handle: "aly.codes", Repo: "tg"}
	if got := target.String(); got != "aly.codes/tg" {
		t.Fatalf("String() = %q, want %q", got, "aly.codes/tg")
	}
}
