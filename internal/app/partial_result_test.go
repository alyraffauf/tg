package app

import (
	"context"
	"errors"
	"testing"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

// methodFailingKnot fails selected Knot procedures after recording them, so a
// multi-step write can succeed in its first step and fail in a later one.
type methodFailingKnot struct {
	*testKnot
	failures map[string]error
}

func (k *methodFailingKnot) Procedure(ctx context.Context, method string, input, out any) error {
	if err := k.testKnot.Procedure(ctx, method, input, out); err != nil {
		return err
	}
	return k.failures[method]
}

func countMethod(methods []string, want string) int {
	count := 0
	for _, method := range methods {
		if method == want {
			count++
		}
	}
	return count
}

func TestForkReturnsRetainedRepositoryWhenDeclarationFails(t *testing.T) {
	rejected := &atclient.APIError{StatusCode: 400, Name: "InvalidRequest"}
	for _, test := range []struct {
		name         string
		failure      error
		cleanupErr   error
		wantDeletes  int
		wantRetained bool
	}{
		{"rejected and deleted", rejected, nil, 1, false},
		{"rejected but cleanup failed", rejected, errors.New("cleanup unavailable"), 1, true},
		{"uncertain declaration write", errors.New("connection lost"), nil, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, pds, origin := migrationService(t)
			pds.getErr = &atclient.APIError{StatusCode: 404, Name: "RecordNotFound"}
			pds.putErr, origin.deleteErr = test.failure, test.cleanupErr

			result, err := service.ForkRepo(context.Background(), migrationTarget(), "fork")

			if !errors.Is(err, test.failure) {
				t.Fatalf("err = %v, want wrapped %v", err, test.failure)
			}
			if origin.deleteCalls != test.wantDeletes {
				t.Fatalf("delete calls = %d, want %d", origin.deleteCalls, test.wantDeletes)
			}
			if (result != nil) != test.wantRetained {
				t.Fatalf("result = %+v, want retained=%t", result, test.wantRetained)
			}
			if result != nil && (result.RepoDID != "did:plc:repo" || result.URI == "" || result.Declared) {
				t.Fatalf("recovery receipt = %+v, want undeclared did:plc:repo", result)
			}
		})
	}
}

func TestCreatePullReportsTicketWhenPatchSubmissionFails(t *testing.T) {
	rejected := &atclient.APIError{StatusCode: 400, Name: "InvalidRequest"}
	ticketURI := "at://did:plc:repo/" + tangled.IssueCollection + "/3m2zq4c6pmk2a"
	for _, test := range []struct {
		name            string
		patchErr        error
		cleanupErr      error
		wantErases      int
		wantPatchStatus string // empty means no ticket is retained
	}{
		{"submitted", nil, nil, 0, PatchSubmitted},
		{"rejected and ticket removed", rejected, nil, 1, ""},
		{"rejected but ticket removal failed", rejected, errors.New("knot offline"), 1, PatchRejected},
		{"uncertain submission", errors.New("response lost"), nil, 0, PatchUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _, origin := migrationService(t)
			client := &methodFailingKnot{testKnot: origin, failures: map[string]error{
				"org.tangled.review.submitPatch": test.patchErr,
				"com.atproto.repo.deleteRecord":  test.cleanupErr,
			}}
			service.knot = &testKnotFactory{client: client}

			result, err := service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Title: "PR"})

			if test.patchErr == nil && err != nil {
				t.Fatalf("err = %v", err)
			}
			if test.patchErr != nil && !errors.Is(err, test.patchErr) {
				t.Fatalf("err = %v, want wrapped %v", err, test.patchErr)
			}
			if erases := countMethod(origin.procedureMethods, "com.atproto.repo.deleteRecord"); erases != test.wantErases {
				t.Fatalf("ticket erasures = %d, want %d", erases, test.wantErases)
			}
			if test.wantPatchStatus == "" {
				if result != nil {
					t.Fatalf("returned deleted ticket as retained: %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("retained ticket missing from result")
			}
			if result.PatchStatus != test.wantPatchStatus || result.URI != ticketURI || result.CID == "" {
				t.Fatalf("result = %+v, want patch status %q with ticket receipt", result, test.wantPatchStatus)
			}
		})
	}
}
