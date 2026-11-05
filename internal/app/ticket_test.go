package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestTicketOperationsUseDirectLookupAndRejectUnexpectedRecords(t *testing.T) {
	const key = "1"
	operations := []struct {
		name string
		pull bool
		run  func(context.Context, *Service) error
	}{
		{"view issue", false, func(ctx context.Context, s *Service) error {
			_, err := s.ViewIssue(ctx, migrationTarget(), key)
			return err
		}},
		{"comment issue", false, func(ctx context.Context, s *Service) error {
			_, err := s.CommentIssue(ctx, migrationTarget(), key, "Comment")
			return err
		}},
		{"view pull", true, func(ctx context.Context, s *Service) error {
			_, err := s.ViewPull(ctx, migrationTarget(), key)
			return err
		}},
		{"comment pull", true, func(ctx context.Context, s *Service) error {
			_, err := s.CommentPull(ctx, migrationTarget(), key, "Comment")
			return err
		}},
		{"pull patch", true, func(ctx context.Context, s *Service) error {
			_, err := s.PullPatch(ctx, migrationTarget(), key)
			return err
		}},
		{"update pull", true, func(ctx context.Context, s *Service) error {
			_, err := s.UpdatePullRound(ctx, UpdatePullRoundInput{Target: migrationTarget(), Rkey: key})
			return err
		}},
		{"merge pull", true, func(ctx context.Context, s *Service) error {
			result, err := s.MergePull(ctx, migrationTarget(), key)
			if err == nil && !result.StatusRecorded {
				return errors.New("merge did not complete the ticket")
			}
			return err
		}},
	}
	missing := &atclient.APIError{StatusCode: 404, Name: "NotFound"}
	for _, operation := range operations {
		for _, response := range []string{"valid", "different URI", "wrong kind", "wrong type", "malformed", "empty", "missing"} {
			t.Run(operation.name+"/"+response, func(t *testing.T) {
				service, _, origin := migrationService(t)
				item := ticketList(operation.pull).Items[0]
				item.URI = "at://did:plc:repo/" + tangled.IssueCollection + "/" + key
				var lookupErr error
				switch response {
				case "different URI":
					item.URI = strings.Replace(item.URI, "did:plc:repo", "did:plc:other", 1)
				case "wrong kind":
					item.Value = ticketList(!operation.pull).Items[0].Value
				case "wrong type":
					item.Value = json.RawMessage(`{"$type":"org.tangled.snippet.string"}`)
				case "malformed":
					item.Value = json.RawMessage(`{"title":7}`)
				case "missing":
					lookupErr = missing
				}
				view := service.appview.(testAppview)
				lookups := 0
				view.getTicketHook = func(uri string) (*tangled.ListItem, error) {
					lookups++
					if uri != "at://did:plc:repo/"+tangled.IssueCollection+"/"+key {
						t.Errorf("unexpected lookup URI: %s", uri)
					}
					if response == "empty" {
						return nil, nil
					}
					return &item, lookupErr
				}
				merged := false
				view.mergeHook = func(string, any) { merged = true }
				service.appview = view
				origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/" + key: {URI: item.URI, CID: cidPointer(), Value: ticketRecord("PR", "", "main")}}
				err := operation.run(context.Background(), service)
				if (err == nil) != (response == "valid") || lookups != 1 {
					t.Fatalf("err=%v lookups=%d", err, lookups)
				}
				if response == "missing" && !errors.Is(err, missing) {
					t.Fatalf("lost lookup error: %v", err)
				}
				if response != "valid" && (merged || len(origin.procedureMethods) != 0) {
					t.Fatalf("unexpected mutation: merged=%v methods=%v", merged, origin.procedureMethods)
				}
			})
		}
	}
}

func TestFosterReadDoesNotChangeTheAppviewSnapshot(t *testing.T) {
	service, _, origin := migrationService(t)
	service.fosterKnot = "foster.example"
	origin.legacy = true
	item := ticketList(false).Items[0]
	view := service.appview.(testAppview)
	view.ticket = &item
	service.appview = view
	key := extractRKey(item.URI)
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/" + key: {URI: item.URI, Value: ticketRecord("Foster title", "", "")}}
	first, err := service.ViewIssue(context.Background(), migrationTarget(), key)
	if err != nil || first.Title != "Foster title" || first.CID != "" {
		t.Fatalf("foster read=%+v err=%v", first, err)
	}
	origin.legacy = false
	second, err := service.ViewIssue(context.Background(), migrationTarget(), key)
	if err != nil || second.Title != "Example" || second.CID != testCID {
		t.Fatalf("upgraded read=%+v err=%v", second, err)
	}
}

func TestCommentPullRejectsPatchFromAnotherRepository(t *testing.T) {
	service, _, origin := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	view.patches = []tangled.Patch{{URI: "at://did:plc:other/org.tangled.review.patch/patch", CID: testCID}}
	service.appview = view
	_, err := service.CommentPull(context.Background(), migrationTarget(), "3m2zq4c6pmk2a", "Comment")
	if err == nil || !strings.Contains(err.Error(), "different repository") || len(origin.procedureMethods) != 0 {
		t.Fatalf("err=%v writes=%v", err, origin.procedureMethods)
	}
}

func TestMergePullRequiresTicketVersionBeforeMerging(t *testing.T) {
	for _, cid := range []string{"", "invalid"} {
		t.Run("cid="+cid, func(t *testing.T) {
			service, _, origin := migrationService(t)
			item := ticketList(true).Items[0]
			item.CID = cid
			view := service.appview.(testAppview)
			view.ticket = &item
			view.mergeHook = func(string, any) { t.Fatal("merged without a ticket version") }
			service.appview = view
			_, err := service.MergePull(context.Background(), migrationTarget(), extractRKey(item.URI))
			if err == nil || !strings.Contains(err.Error(), "valid CID") || len(origin.procedureMethods) != 0 {
				t.Fatalf("err=%v writes=%v", err, origin.procedureMethods)
			}
		})
	}
}

func TestTicketReadRejectsInvalidKeyBeforeLookup(t *testing.T) {
	service, _, _ := migrationService(t)
	view := service.appview.(testAppview)
	view.getTicketHook = func(string) (*tangled.ListItem, error) {
		t.Fatal("invalid record key reached the appview")
		return nil, nil
	}
	service.appview = view
	if _, err := service.ViewIssue(context.Background(), migrationTarget(), "ticket/other"); err == nil {
		t.Fatal("accepted an invalid record key")
	}
}

func TestTicketListsWarnOnWrongResourceKind(t *testing.T) {
	for _, pull := range []bool{false, true} {
		t.Run(map[bool]string{false: "issues", true: "pulls"}[pull], func(t *testing.T) {
			service, _, _ := migrationService(t)
			view := service.appview.(testAppview)
			view.issues = ticketList(true)
			view.pulls = ticketList(false)
			service.appview = view
			var result *ItemListResult
			var err error
			if pull {
				result, err = service.ListPulls(context.Background(), migrationTarget(), ListOptions{})
			} else {
				result, err = service.ListIssues(context.Background(), migrationTarget(), ListOptions{})
			}
			if err != nil || len(result.Items) != 0 || len(result.Warnings) != 1 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestMergeRejectsStaleOrClosedAuthoritativeTicket(t *testing.T) {
	for _, state := range []string{"open", "closed"} {
		t.Run(state, func(t *testing.T) {
			service, _, origin := migrationService(t)
			item := ticketList(true).Items[0]
			view := service.appview.(testAppview)
			view.ticket = &item
			view.mergeHook = func(string, any) { t.Fatal("merged a stale or closed ticket") }
			service.appview = view
			record := ticketRecord("PR", "", "main")
			record.State = state
			origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/" + extractRKey(item.URI): {URI: item.URI, CID: cidPointer(), Value: record}}
			if state == "open" {
				item.CID = "bafyreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"
			}
			if _, err := service.MergePull(context.Background(), migrationTarget(), extractRKey(item.URI)); err == nil {
				t.Fatal("accepted stale or closed ticket")
			}
			if len(origin.procedureMethods) != 0 {
				t.Fatal("modified ticket after rejected merge")
			}
		})
	}
}
