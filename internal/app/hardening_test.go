package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestCreateRepoDeletesKnotRepoWhenRecordWriteFails(t *testing.T) {
	pds := &testPDS{putErr: errors.New("PDS unavailable")}
	knotClient := &testKnot{}
	service := testService(pds, &testGit{}, knotClient)

	_, err := service.CreateRepo(context.Background(), CreateRepoInput{KnotHost: "knot.example", Name: "example"})
	if err == nil || !strings.Contains(err.Error(), "write repository record") {
		t.Fatalf("CreateRepo() error = %v", err)
	}
	if knotClient.createCalls != 1 || knotClient.deleteCalls != 1 {
		t.Fatalf("Knot calls: create=%d delete=%d", knotClient.createCalls, knotClient.deleteCalls)
	}
	if len(pds.serviceAuthLexiconMethods) != 2 || pds.serviceAuthLexiconMethods[1] != "sh.tangled.repo.delete" {
		t.Fatalf("service auth methods = %q", pds.serviceAuthLexiconMethods)
	}
}

func TestCreateRepoReportsRecordAndCleanupFailures(t *testing.T) {
	pds := &testPDS{putErr: errors.New("PDS unavailable")}
	knotClient := &testKnot{deleteErr: errors.New("Knot cleanup unavailable")}
	service := testService(pds, &testGit{}, knotClient)

	_, err := service.CreateRepo(context.Background(), CreateRepoInput{KnotHost: "knot.example", Name: "example"})
	if err == nil || !strings.Contains(err.Error(), "PDS unavailable") || !strings.Contains(err.Error(), "Knot cleanup unavailable") {
		t.Fatalf("CreateRepo() error = %v", err)
	}
}

func TestCreateRepoRejectsInvalidRecordKeyBeforeDependencies(t *testing.T) {
	pds := &testPDS{}
	knotClient := &testKnot{}
	service := testService(pds, &testGit{}, knotClient)

	_, err := service.CreateRepo(context.Background(), CreateRepoInput{Name: "nested/repo"})
	if err == nil || !strings.Contains(err.Error(), "invalid repository name") {
		t.Fatalf("CreateRepo() error = %v", err)
	}
	if knotClient.createCalls != 0 || pds.serviceAuthCalls != 0 {
		t.Fatalf("dependencies called after validation failure: knot=%d auth=%d", knotClient.createCalls, pds.serviceAuthCalls)
	}
}

func TestDeleteRepoStopsOnTransientRecordFetchFailure(t *testing.T) {
	pds := &testPDS{getErr: errors.New("PDS unavailable")}
	knotClient := &testKnot{}
	service := testService(pds, &testGit{}, knotClient)
	service.appview = ownedRepoAppview()

	_, err := service.DeleteRepo(context.Background(), Target{Handle: "owner.test", Repo: "example"})
	if err == nil || !strings.Contains(err.Error(), "fetch repository record") {
		t.Fatalf("DeleteRepo() error = %v", err)
	}
	if len(pds.deletes) != 0 || knotClient.deleteCalls != 0 || pds.serviceAuthCalls != 0 {
		t.Fatalf("mutations after fetch failure: PDS=%d Knot=%d auth=%d", len(pds.deletes), knotClient.deleteCalls, pds.serviceAuthCalls)
	}
}

func TestDeleteRepoContinuesOnlyForRecordNotFound(t *testing.T) {
	pds := &testPDS{getErr: &atclient.APIError{StatusCode: http.StatusNotFound, Name: "RecordNotFound"}}
	knotClient := &testKnot{}
	service := testService(pds, &testGit{}, knotClient)
	service.appview = ownedRepoAppview()

	if _, err := service.DeleteRepo(context.Background(), Target{Handle: "owner.test", Repo: "example"}); err != nil {
		t.Fatalf("DeleteRepo() error = %v", err)
	}
	if len(pds.deletes) != 0 || knotClient.deleteCalls != 1 {
		t.Fatalf("mutations: PDS=%d Knot=%d", len(pds.deletes), knotClient.deleteCalls)
	}
}

func TestStateChangesRejectUnsupportedValuesBeforeLookup(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	for name, call := range map[string]func() error{
		"issue": func() error {
			_, err := service.SetIssueState(context.Background(), Target{}, "one", "future")
			return err
		},
		"pull": func() error {
			_, err := service.SetPullState(context.Background(), Target{}, "one", "merged")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Fatal("state change accepted unsupported value")
			}
		})
	}
	if pds.getCalls != 0 || len(pds.puts) != 0 {
		t.Fatalf("PDS calls after validation failure: get=%d put=%d", pds.getCalls, len(pds.puts))
	}
}

func ownedRepoAppview() testAppview {
	return testAppview{repo: &tangled.Repo{
		URI: "at://did:plc:owner/sh.tangled.repo/example",
		Value: tangledlex.Repo{
			LexiconTypeID: repoCollection, Knot: "knot.example", CreatedAt: "2026-07-25T12:00:00Z",
			RepoDid: optionalString("did:plc:repo"),
		},
	}}
}
