package app

import (
	"context"
	"strings"
	"testing"
)

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
