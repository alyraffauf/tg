package app

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
)

const labelTicketKey = "3m2zq4c6pmk2a"

func labelDefinition(key, name, color string) tangled.LabelDefinition {
	return tangled.LabelDefinition{URI: "at://did:plc:repo/" + tangled.LabelDefinitionCollection + "/" + key, Key: key, Name: name, Color: color}
}

func labelToken(key string) map[string]any {
	return map[string]any{"$type": tangled.LabelTokenType, "key": key}
}

// labelService serves the given definitions and stores one ticket on the knot.
func labelService(t *testing.T, definitions []tangled.LabelDefinition, ticket map[string]any) (*Service, *testKnot, *int) {
	t.Helper()
	service, _, origin := migrationService(t)
	definitionReads := 0
	view := service.appview.(testAppview)
	view.labelDefinitionsHook = func(repoDID string) (*tangled.LabelDefinitionList, error) {
		definitionReads++
		if repoDID != "did:plc:repo" {
			t.Errorf("definitions read for %q", repoDID)
		}
		return &tangled.LabelDefinitionList{Items: definitions}, nil
	}
	service.appview = view
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/" + labelTicketKey: {
		URI: "at://did:plc:repo/" + tangled.IssueCollection + "/" + labelTicketKey, CID: cidPointer(), Value: ticket,
	}}
	return service, origin, &definitionReads
}

func storedTicket(labels ...any) map[string]any {
	ticket := map[string]any{
		"$type":     tangled.IssueCollection,
		"title":     map[string]any{"$type": "org.tangled.markup.markdown#inline", "text": "Original"},
		"body":      map[string]any{"$type": "org.tangled.markup.markdown", "text": "Body"},
		"state":     "open",
		"assignees": []any{"did:plc:alice"},
		"createdAt": "2026-10-01T00:00:00Z",
	}
	if len(labels) > 0 {
		ticket["tangledLabels"] = labels
	}
	return ticket
}

func savedRecord(t *testing.T, origin *testKnot) map[string]any {
	t.Helper()
	if len(origin.procedureInputs) != 1 || origin.procedureMethods[0] != "com.atproto.repo.putRecord" {
		t.Fatalf("writes=%v", origin.procedureMethods)
	}
	if origin.procedureInputs[0]["swapRecord"] != testCID {
		t.Fatalf("write did not swap the read version: %+v", origin.procedureInputs[0])
	}
	return origin.procedureInputs[0]["record"].(map[string]any)
}

func TestLabelEditKeepsOtherFieldsAndDropsOrphanedLabels(t *testing.T) {
	definitions := []tangled.LabelDefinition{labelDefinition("bug", "bug", "#f00"), labelDefinition("docs", "docs", "")}
	service, origin, _ := labelService(t, definitions, storedTicket(labelToken("bug"), labelToken("deleted-definition")))

	result, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Labels: LabelEditInput{AddLabels: []string{"docs"}}})
	if err != nil {
		t.Fatal(err)
	}
	saved := savedRecord(t, origin)
	wantLabels := []any{labelToken("bug"), labelToken("docs")}
	if !reflect.DeepEqual(saved["tangledLabels"], wantLabels) {
		t.Fatalf("labels=%v, want %v", saved["tangledLabels"], wantLabels)
	}
	for field, want := range map[string]any{"assignees": []any{"did:plc:alice"}, "state": "open", "createdAt": "2026-10-01T00:00:00Z"} {
		if !reflect.DeepEqual(saved[field], want) {
			t.Fatalf("%s=%v, want %v", field, saved[field], want)
		}
	}
	if saved["title"].(map[string]any)["text"] != "Original" || saved["body"].(map[string]any)["text"] != "Body" {
		t.Fatalf("title or body changed: %+v", saved)
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "dropped applied label definitions not found: deleted-definition") {
		t.Fatalf("no warning for the unresolved label: %v", result.Warnings)
	}
}

func TestTitleEditKeepsAppliedLabelsWithoutReadingDefinitions(t *testing.T) {
	service, origin, definitionReads := labelService(t, nil, storedTicket(labelToken("bug")))
	title := "Renamed"
	if _, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Title: &title}); err != nil {
		t.Fatal(err)
	}
	saved := savedRecord(t, origin)
	if !reflect.DeepEqual(saved["tangledLabels"], []any{labelToken("bug")}) {
		t.Fatalf("labels=%v", saved["tangledLabels"])
	}
	if *definitionReads != 0 {
		t.Fatalf("title edit read label definitions %d times", *definitionReads)
	}
}

func TestEditsAcceptKnotKeyOnlyLabelTokens(t *testing.T) {
	for _, action := range []string{"remove", "add", "title"} {
		t.Run(action, func(t *testing.T) {
			// The pinned Knot serializer emits {"key":"bug"} without $type.
			service, origin, _ := labelService(t, []tangled.LabelDefinition{
				labelDefinition("bug", "bug", ""), labelDefinition("docs", "docs", ""),
			}, storedTicket(map[string]any{"key": "bug"}))
			var title *string
			edit := LabelEditInput{}
			var want any
			switch action {
			case "remove":
				edit.RemoveLabels = []string{"bug"}
			case "add":
				edit.AddLabels = []string{"docs"}
				want = []any{labelToken("bug"), labelToken("docs")}
			case "title":
				text := "Renamed"
				title = &text
				want = []any{labelToken("bug")}
			}
			if _, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Title: title, Labels: edit}); err != nil {
				t.Fatal(err)
			}
			if got := savedRecord(t, origin)["tangledLabels"]; !reflect.DeepEqual(got, want) {
				t.Fatalf("labels=%v, want %v", got, want)
			}
		})
	}
}

func TestReadLabelsAcceptsKeyOnlyTokensAndIgnoresUnknownTypes(t *testing.T) {
	keys := appliedLabelKeys([]tangled.LabelToken{
		{Key: "bug"},
		{Type: "org.example.unknown", Key: "docs"},
	})
	if !reflect.DeepEqual(keys, []string{"bug"}) {
		t.Fatalf("keys=%v", keys)
	}
}

func TestPullLabelEditRemovesAndDeduplicates(t *testing.T) {
	definitions := []tangled.LabelDefinition{labelDefinition("bug", "bug", ""), labelDefinition("docs", "docs", "")}
	tests := []struct {
		name   string
		stored []any
		edit   LabelEditInput
		want   any
	}{
		{"removing the last label omits the field", []any{labelToken("bug")}, LabelEditInput{RemoveLabels: []string{"bug"}}, nil},
		{"replacing keeps one token per label", []any{labelToken("bug"), labelToken("docs"), labelToken("docs")}, LabelEditInput{RemoveLabels: []string{"bug"}, AddLabels: []string{"docs"}}, []any{labelToken("docs")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ticket := storedTicket(test.stored...)
			ticket["targetBranch"] = "main"
			service, origin, _ := labelService(t, definitions, ticket)
			if _, err := service.EditPull(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Labels: test.edit}); err != nil {
				t.Fatal(err)
			}
			saved := savedRecord(t, origin)
			if got, present := saved["tangledLabels"]; test.want == nil && present || test.want != nil && !reflect.DeepEqual(got, test.want) {
				t.Fatalf("labels=%v present=%v, want %v", got, present, test.want)
			}
		})
	}
}

func TestLabelEditWithoutChangesDoesNotWrite(t *testing.T) {
	service, origin, _ := labelService(t, []tangled.LabelDefinition{labelDefinition("bug", "bug", ""), labelDefinition("docs", "docs", "")}, storedTicket(labelToken("bug")))
	result, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Labels: LabelEditInput{AddLabels: []string{"bug"}, RemoveLabels: []string{"docs"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatalf("idempotent edit wrote: %v", origin.procedureMethods)
	}
	if result.CID != testCID || result.Rkey != labelTicketKey {
		t.Fatalf("result=%+v", result)
	}
}

func TestLabelEditRejectsUnresolvableNamesBeforeWriting(t *testing.T) {
	definitions := []tangled.LabelDefinition{labelDefinition("bug", "bug", ""), labelDefinition("bug-2", "bug", ""), labelDefinition("docs", "docs", "")}
	tests := []struct {
		name string
		edit LabelEditInput
		want error
	}{
		{"unknown", LabelEditInput{AddLabels: []string{"Docs"}}, ErrUnknownLabel},
		{"ambiguous", LabelEditInput{RemoveLabels: []string{"bug"}}, ErrAmbiguousLabel},
		{"added and removed", LabelEditInput{AddLabels: []string{"docs"}, RemoveLabels: []string{"docs"}}, ErrConflictingLabelEdit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, origin, _ := labelService(t, definitions, storedTicket())
			title := "Also renamed"
			_, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Title: &title, Labels: test.edit})
			if !errors.Is(err, test.want) {
				t.Fatalf("err=%v, want %v", err, test.want)
			}
			if len(origin.procedureMethods) != 0 {
				t.Fatalf("rejected edit wrote: %v", origin.procedureMethods)
			}
		})
	}
}

func TestEditWithoutChangesIsRejected(t *testing.T) {
	service, origin, _ := labelService(t, nil, storedTicket())
	if _, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Labels: LabelEditInput{}}); err == nil {
		t.Fatal("accepted an edit with no changes")
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatalf("empty edit wrote: %v", origin.procedureMethods)
	}
}

func listedTicket(t *testing.T, key string, labelKeys ...string) tangled.ListItem {
	t.Helper()
	record := ticketRecord("Ticket "+key, "", "")
	for _, labelKey := range labelKeys {
		record.TangledLabels = append(record.TangledLabels, tangled.LabelToken{Type: tangled.LabelTokenType, Key: labelKey})
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return tangled.ListItem{URI: "at://did:plc:repo/" + tangled.IssueCollection + "/" + key, Value: data, Author: "did:plc:alice", State: "open"}
}

func TestLabelFilterMatchesEveryNameAndForwardsLimit(t *testing.T) {
	for _, pull := range []bool{false, true} {
		t.Run(ticketKindPlural(pull), func(t *testing.T) {

			// "bug" names two definitions; a filter accepts either of them.
			definitions := []tangled.LabelDefinition{labelDefinition("bug", "bug", "#f00"), labelDefinition("bug-2", "bug", ""), labelDefinition("docs", "docs", "")}
			service, _, _ := labelService(t, definitions, storedTicket())
			view := service.appview.(testAppview)
			view.issues = &tangled.List{Items: []tangled.ListItem{
				listedTicket(t, "unlabeled"),
				listedTicket(t, "bug-only", "bug"),
				listedTicket(t, "first-match", "docs", "bug-2"),
				listedTicket(t, "second-match", "bug", "docs"),
			}}
			hook := func(options tangled.ListOpts) (*tangled.List, error) {
				if options.MaxItems != 1 || options.Match == nil {
					t.Fatalf("filter options=%+v", options)
				}
				// Fixed expectations exercise all-name matching and ambiguous names.
				// Pagination and limit enforcement belong to the real client's tests.
				for index, want := range []bool{false, false, true, true} {
					var ticket tangled.Ticket
					if err := json.Unmarshal(view.issues.Items[index].Value, &ticket); err != nil {
						t.Fatal(err)
					}
					if got := options.Match(ticket); got != want {
						t.Errorf("match for %s=%v, want %v", view.issues.Items[index].URI, got, want)
					}
				}
				return &tangled.List{Items: view.issues.Items[2:3]}, nil
			}
			if pull {
				for index := range view.issues.Items {
					var ticket tangled.Ticket
					if err := json.Unmarshal(view.issues.Items[index].Value, &ticket); err != nil {
						t.Fatal(err)
					}
					ticket.TargetBranch = "main"
					data, err := json.Marshal(ticket)
					if err != nil {
						t.Fatal(err)
					}
					view.issues.Items[index].Value = data
				}
				view.listPullsHook = hook
			} else {
				view.listIssuesHook = hook
			}
			service.appview = view
			list := service.ListIssues
			if pull {
				list = service.ListPulls
			}

			result, err := list(context.Background(), migrationTarget(), ListOptions{Labels: []string{"bug", "docs"}, Limit: 1})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Items) != 1 || result.Items[0].Rkey != "first-match" {
				t.Fatalf("items=%+v", result.Items)
			}
			wantLabels := []Label{{Name: "docs", URI: definitions[2].URI}, {Name: "bug", URI: definitions[1].URI}}
			if !reflect.DeepEqual(result.Items[0].Labels, wantLabels) {
				t.Fatalf("labels=%+v, want %+v", result.Items[0].Labels, wantLabels)
			}

			if _, err := list(context.Background(), migrationTarget(), ListOptions{Labels: []string{"wontfix"}}); !errors.Is(err, ErrUnknownLabel) {
				t.Fatalf("unknown filter err=%v", err)
			}

		})
	}
}

func TestListingLabelsWithoutDefinitions(t *testing.T) {
	service, _, _ := migrationService(t)
	view := service.appview.(testAppview)
	view.issues = &tangled.List{Items: []tangled.ListItem{listedTicket(t, "unlabeled"), listedTicket(t, "labeled", "bug")}}
	view.labelDefinitionsHook = func(string) (*tangled.LabelDefinitionList, error) { return nil, errors.New("appview unavailable") }
	service.appview = view

	result, err := service.ListIssues(context.Background(), migrationTarget(), ListOptions{})
	if err != nil {
		t.Fatalf("display listing failed without definitions: %v", err)
	}
	encoded, err := json.Marshal(result.Items[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"labels":[]`) {
		t.Fatalf("unlabeled item JSON=%s", encoded)
	}
	if got := result.Items[1].Labels; len(got) != 1 || got[0].URI != "at://did:plc:repo/"+tangled.LabelDefinitionCollection+"/bug" {
		t.Fatalf("labels without definitions=%+v", got)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings=%+v, want one for the unavailable definitions", result.Warnings)
	}

	if _, err := service.ListIssues(context.Background(), migrationTarget(), ListOptions{Labels: []string{"bug"}}); err == nil {
		t.Fatal("label filter succeeded without definitions")
	}
}

func TestViewsNameLabels(t *testing.T) {
	definitions := []tangled.LabelDefinition{labelDefinition("bug", "bug", "#f00"), labelDefinition("docs", "docs", "")}
	service, _, _ := labelService(t, definitions, storedTicket())
	view := service.appview.(testAppview)
	item := listedTicket(t, labelTicketKey, "deleted-definition", "bug")
	item.URI = "at://did:plc:repo/" + tangled.IssueCollection + "/" + labelTicketKey
	view.ticket = &item
	service.appview = view

	issue, err := service.ViewIssue(context.Background(), migrationTarget(), labelTicketKey)
	if err != nil {
		t.Fatal(err)
	}
	wantIssueLabels := []Label{{URI: "at://did:plc:repo/" + tangled.LabelDefinitionCollection + "/deleted-definition"}, {Name: "bug", Color: "#f00", URI: definitions[0].URI}}
	if !reflect.DeepEqual(issue.Labels, wantIssueLabels) {
		t.Fatalf("issue labels=%+v, want %+v", issue.Labels, wantIssueLabels)
	}
	if !strings.Contains(strings.Join(issue.Warnings, "\n"), "deleted-definition") {
		t.Fatalf("no warning for the unresolved label: %v", issue.Warnings)
	}

	repo, err := service.ViewRepo(context.Background(), migrationTarget())
	if err != nil {
		t.Fatal(err)
	}
	wantRepoLabels := []Label{{Name: "bug", Color: "#f00", URI: definitions[0].URI}, {Name: "docs", URI: definitions[1].URI}}
	if !reflect.DeepEqual(repo.Labels, wantRepoLabels) {
		t.Fatalf("repo labels=%+v, want %+v", repo.Labels, wantRepoLabels)
	}
}

func TestRepoJSONLabelsDistinguishViewsFromListings(t *testing.T) {
	service, _, _ := labelService(t, nil, storedTicket())
	viewed, err := service.ViewRepo(context.Background(), migrationTarget())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		item RepoItem
		want string
	}{{*viewed, `"labels":[]`}, {RepoItem{Name: "listed"}, ""}} {
		encoded, err := json.Marshal(test.item)
		if err != nil {
			t.Fatal(err)
		}
		hasLabels := strings.Contains(string(encoded), `"labels"`)
		if test.want == "" && hasLabels || test.want != "" && !strings.Contains(string(encoded), test.want) {
			t.Fatalf("JSON=%s, want labels %q", encoded, test.want)
		}
	}
}

func TestIncompleteDefinitionsBlockNameBasedOperations(t *testing.T) {
	service, origin, _ := labelService(t, nil, storedTicket())
	view := service.appview.(testAppview)
	view.labelDefinitionsHook = func(string) (*tangled.LabelDefinitionList, error) {
		// A malformed duplicate may have been skipped, so "bug" is not known unique.
		return &tangled.LabelDefinitionList{
			Items:    []tangled.LabelDefinition{labelDefinition("bug", "bug", "")},
			Warnings: []tangled.RecordWarning{{Error: "invalid definition"}},
		}, nil
	}
	service.appview = view
	if _, err := service.EditIssue(context.Background(), EditTicketInput{
		Target: migrationTarget(), Rkey: labelTicketKey, Labels: LabelEditInput{AddLabels: []string{"bug"}},
	}); !errors.Is(err, ErrIncompleteLabelCatalog) {
		t.Fatalf("edit error=%v", err)
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatalf("writes=%v", origin.procedureMethods)
	}
	for _, list := range []func(context.Context, Target, ListOptions) (*ItemListResult, error){service.ListIssues, service.ListPulls} {
		if _, err := list(context.Background(), migrationTarget(), ListOptions{Labels: []string{"bug"}}); !errors.Is(err, ErrIncompleteLabelCatalog) {
			t.Fatalf("filter error=%v", err)
		}
	}
	repo, err := service.ViewRepo(context.Background(), migrationTarget())
	if err != nil || len(repo.Labels) != 1 || len(repo.Warnings) == 0 {
		t.Fatalf("partial display=%+v, error=%v", repo, err)
	}
}

func TestUnlabeledListsDoNotReadDefinitions(t *testing.T) {
	for _, pull := range []bool{false, true} {
		t.Run(ticketKindPlural(pull), func(t *testing.T) {
			service, _, reads := labelService(t, nil, storedTicket())
			view := service.appview.(testAppview)
			item := listedTicket(t, "unlabeled")
			if pull {
				var ticket tangled.Ticket
				if err := json.Unmarshal(item.Value, &ticket); err != nil {
					t.Fatal(err)
				}
				ticket.TargetBranch = "main"
				data, err := json.Marshal(ticket)
				if err != nil {
					t.Fatal(err)
				}
				item.Value = data
			}
			view.issues = &tangled.List{Items: []tangled.ListItem{item}}
			view.pulls = view.issues
			service.appview = view
			list := service.ListIssues
			if pull {
				list = service.ListPulls
			}
			result, err := list(context.Background(), migrationTarget(), ListOptions{})
			if err != nil || len(result.Items) != 1 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if *reads != 0 || len(result.Warnings) != 0 {
				t.Fatalf("reads=%d warnings=%v", *reads, result.Warnings)
			}
		})
	}
}

func TestUnknownLabelTokenTypeRejectsEditsWithoutWriting(t *testing.T) {
	service, origin, _ := labelService(t, nil, storedTicket(map[string]any{"$type": "org.example.unknown", "key": "bug"}))
	title := "Changed"
	if _, err := service.EditIssue(context.Background(), EditTicketInput{Target: migrationTarget(), Rkey: labelTicketKey, Title: &title}); err == nil {
		t.Fatal("edit accepted an unknown label type")
	}
	if len(origin.procedureMethods) != 0 {
		t.Fatalf("writes=%v", origin.procedureMethods)
	}
}
