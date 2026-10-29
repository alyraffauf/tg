package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestRecordNotFoundResponses(t *testing.T) {
	for _, tc := range []struct {
		status  int
		name    string
		missing bool
	}{
		{400, "RecordNotFound", true},
		{404, "RecordNotFound", true},
		{404, "NotFound", true},
		{400, "NotFound", false},
		{401, "RecordNotFound", false},
		{500, "RecordNotFound", false},
		{400, "InvalidRequest", false},
	} {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.name), func(t *testing.T) {
			err := fmt.Errorf("fetch: %w", &atclient.APIError{StatusCode: tc.status, Name: tc.name})
			if got := isRecordNotFound(err); got != tc.missing {
				t.Fatalf("isRecordNotFound() = %v, want %v", got, tc.missing)
			}
		})
	}
}

func TestReadSurvivesUnavailableOwnerPDSButWriteDoesNot(t *testing.T) {
	s, pds, _ := migrationService(t)
	pds.getErr = errors.New("owner PDS offline")
	if _, err := s.ViewRepo(context.Background(), migrationTarget()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateIssue(context.Background(), migrationTarget(), "Title", ""); err == nil {
		t.Fatal("write accepted without declaration verification")
	}
}

func TestForkReportsOneImmediateStatus(t *testing.T) {
	for _, status := range []string{"fetching", "present", "partial", "failed", ""} {
		t.Run(status, func(t *testing.T) {
			s, pds, origin := migrationService(t)
			pds.getErr = &atclient.APIError{StatusCode: 404, Name: "RecordNotFound"}
			origin.importStatus = &knot.ImportStatus{Status: status, Reason: "source detail"}
			result, err := s.ForkRepo(context.Background(), migrationTarget(), "fork")
			if err != nil {
				t.Fatal(err)
			}
			if result.ImportStatus != status || origin.importCalls != 1 {
				t.Fatalf("%+v calls=%d", result, origin.importCalls)
			}
			if status != "present" && len(result.Warnings) == 0 {
				t.Fatal("missing incomplete import warning")
			}
		})
	}
	s, pds, origin := migrationService(t)
	pds.getErr = &atclient.APIError{StatusCode: 404, Name: "RecordNotFound"}
	origin.importErr = errors.New("offline")
	result, err := s.ForkRepo(context.Background(), migrationTarget(), "fork")
	if err != nil || result.URI == "" || len(result.Warnings) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestExpectedCIDRejectsBeforeMutation(t *testing.T) {
	s, _, origin := migrationService(t)
	title := "Changed"
	if _, err := s.EditIssue(context.Background(), migrationTarget(), "ticket", &title, nil, "stale"); err == nil {
		t.Fatal("stale edit accepted")
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatal("stale edit reached knot")
	}
}

func TestFosterViewNeverPairsNewBodyWithOldCID(t *testing.T) {
	s, _, origin := migrationService(t)
	origin.legacy = true
	s.fosterKnot = "foster.example"
	item := ticketList(false).Items[0]
	view := s.appview.(testAppview)
	view.ticket = &item
	s.appview = view
	key := extractRKey(item.URI)
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/" + key: {URI: item.URI, Value: ticketRecord("Foster version", "", "")}}
	result, err := s.ViewIssue(context.Background(), migrationTarget(), key)
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Foster version" || result.CID != "" {
		t.Fatalf("mismatched version: %+v", result)
	}
}

func TestTicketViewRejectsMismatchedIdentity(t *testing.T) {
	s, _, _ := migrationService(t)
	item := ticketList(false).Items[0]
	view := s.appview.(testAppview)
	view.ticket = &item
	s.appview = view
	if _, err := s.ViewIssue(context.Background(), migrationTarget(), "another-ticket"); err == nil {
		t.Fatal("accepted a different ticket")
	}
}

func TestExpectedCIDSuccessReturnsUsableReceipt(t *testing.T) {
	s, _, _ := migrationService(t)
	title := "Changed"
	result, err := s.EditIssue(context.Background(), migrationTarget(), "ticket", &title, nil, testCID)
	if err != nil {
		t.Fatal(err)
	}
	if result.URI == "" || result.CID == "" {
		t.Fatalf("missing record reference: %+v", result)
	}
}

func TestEditRejectsReceiptForDifferentRecord(t *testing.T) {
	s, _, origin := migrationService(t)
	origin.receipt = knot.Receipt{URI: "at://did:plc:repo/org.tangled.track.ticket/another-ticket", CID: testCID}
	title := "Changed"
	if _, err := s.EditIssue(context.Background(), migrationTarget(), "ticket", &title, nil); err == nil {
		t.Fatal("accepted receipt for a different ticket")
	}
}
