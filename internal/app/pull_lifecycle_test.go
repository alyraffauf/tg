package app

import (
	"context"
	"errors"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
)

const lifecycleKey = "3m2zq4c6pmk2a"

func TestCheckPullTransition(t *testing.T) {
	for _, test := range []struct {
		current, next string
		allowed       bool
	}{
		{"open", "closed", true},
		{"closed", "open", true},
		{"open", "completed", true},
		{"closed", "completed", false},
		{"completed", "open", false},
		{"completed", "closed", false},
		{"completed", "completed", false},
		{"open", "merged", false},
	} {
		err := checkPullTransition(test.current, test.next)
		if (err == nil) != test.allowed {
			t.Errorf("%s -> %s: err=%v", test.current, test.next, err)
		}
	}
}

// lifecycleService returns a service whose appview snapshot shows an open
// pull request while the authoritative knot record has the given state.
func lifecycleService(t *testing.T, authoritativeState string) (*Service, *testKnot) {
	t.Helper()
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	service.appview = view
	record := ticketRecord("PR", "", "main")
	record.State = authoritativeState
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/" + lifecycleKey: {URI: view.ticket.URI, CID: cidPointer(), Value: record}}
	return service, origin
}

func TestSetPullStateNeverRewritesARecordedMerge(t *testing.T) {
	for _, state := range []string{"open", "closed"} {
		t.Run(state, func(t *testing.T) {
			service, origin := lifecycleService(t, "completed")
			_, err := service.SetPullState(context.Background(), migrationTarget(), lifecycleKey, state)
			if !errors.Is(err, errPullAlreadyMerged) || len(origin.procedureMethods) != 0 {
				t.Fatalf("err=%v writes=%v", err, origin.procedureMethods)
			}
		})
	}
}

func TestPullStateChangesSwapTheCheckedRecord(t *testing.T) {
	service, origin := lifecycleService(t, "open")
	if _, err := service.SetPullState(context.Background(), migrationTarget(), lifecycleKey, "closed"); err != nil {
		t.Fatal(err)
	}
	if len(origin.procedureInputs) != 1 || origin.procedureInputs[0]["swapRecord"] != testCID {
		t.Fatalf("state change was not conditional on the checked record: %+v", origin.procedureInputs)
	}
}

func TestEditingAMergedPullKeepsItsState(t *testing.T) {
	service, origin := lifecycleService(t, "completed")
	title := "Renamed"
	if _, err := service.EditPull(context.Background(), migrationTarget(), lifecycleKey, &title, nil, ""); err != nil {
		t.Fatal(err)
	}
	if state := origin.procedureInputs[0]["record"].(map[string]any)["state"]; state != "completed" {
		t.Fatalf("edit changed state to %v", state)
	}
}
