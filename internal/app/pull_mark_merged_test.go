package app

import (
	"context"
	"errors"
	"testing"
)

func TestMarkPullMergedOnlyUpdatesStatus(t *testing.T) {
	service, origin := lifecycleService(t, "open")
	view := service.appview.(testAppview)
	view.mergeHook = func(string, any) { t.Fatal("manual completion attempted a Git merge") }
	service.appview = view
	result, err := service.MarkPullMerged(context.Background(), MarkPullMergedInput{Target: migrationTarget(), Rkey: lifecycleKey})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "merged" || result.URI == "" || result.CID == "" {
		t.Fatalf("result=%+v", result)
	}
	if len(origin.procedureInputs) != 1 {
		t.Fatalf("writes=%v", origin.procedureInputs)
	}
	update := origin.procedureInputs[0]
	if update["swapRecord"] != testCID || update["record"].(map[string]any)["state"] != "completed" {
		t.Fatalf("unconditional or incorrect state update: %v", update)
	}
}

func TestMarkPullMergedRejectsClosedCompletedOrStaleTicketsWithoutWrites(t *testing.T) {
	for _, test := range []struct{ name, state, expected string }{
		{"closed", "closed", ""},
		{"completed", "completed", ""},
		{"stale", "open", "stale"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, origin := lifecycleService(t, test.state)
			_, err := service.MarkPullMerged(context.Background(), MarkPullMergedInput{Target: migrationTarget(), Rkey: lifecycleKey, ExpectedCID: test.expected})
			if err == nil || len(origin.procedureMethods) != 0 {
				t.Fatalf("err=%v writes=%v", err, origin.procedureMethods)
			}
		})
	}
}

func TestMarkPullMergedReportsStatusWriteFailure(t *testing.T) {
	service, origin := lifecycleService(t, "open")
	origin.procedureErr = errors.New("write denied")
	result, err := service.MarkPullMerged(context.Background(), MarkPullMergedInput{Target: migrationTarget(), Rkey: lifecycleKey})
	if err == nil || result != nil || len(origin.procedureMethods) != 1 {
		t.Fatalf("result=%+v err=%v writes=%v", result, err, origin.procedureMethods)
	}
}
