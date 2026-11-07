package cli

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type labelEditRecorder struct {
	target      app.Target
	title, body *string
	labels      app.LabelEditInput
	expectedCID []string
	calls       int
}

func (recorder *labelEditRecorder) TargetFromCWD(context.Context) (app.Target, error) {
	return app.Target{Handle: "alice.test", Repo: "project"}, nil
}

func (recorder *labelEditRecorder) EditIssue(_ context.Context, input app.EditTicketInput) (*app.UpdatedRecordResult, error) {
	recorder.calls++
	recorder.target, recorder.title, recorder.body = input.Target, input.Title, input.Body
	recorder.labels, recorder.expectedCID = input.Labels, []string{input.ExpectedCID}
	return &app.UpdatedRecordResult{Rkey: "ticket"}, nil
}
func (recorder *labelEditRecorder) EditPull(ctx context.Context, input app.EditTicketInput) (*app.UpdatedRecordResult, error) {
	return recorder.EditIssue(ctx, input)
}

func TestLabelOnlyEdits(t *testing.T) {
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	for _, kind := range []string{"issue", "pull"} {
		for _, withText := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/labels", true: "/labels-and-text"}[withText], func(t *testing.T) {
				recorder := &labelEditRecorder{}
				command := newIssueEditCommand(recorder)
				if kind == "pull" {
					command = newPREditCommand(recorder)
				}
				args := []string{"ticket", "--repo", "bob.test/other", "--add-label", "bug, urgent", "--add-label", "help wanted", "--remove-label", "done", "--expected-cid", "current"}
				if withText {
					args = append(args, "--title", "New title", "--body", "")
				}
				command.SetArgs(args)
				command.SetOut(io.Discard)
				command.SetErr(io.Discard)
				if err := command.Execute(); err != nil {
					t.Fatal(err)
				}
				if recorder.calls != 1 || recorder.target.Handle != "bob.test" || recorder.target.Repo != "other" {
					t.Fatalf("unexpected call: %+v", recorder)
				}
				want := app.LabelEditInput{AddLabels: []string{"bug, urgent", "help wanted"}, RemoveLabels: []string{"done"}}
				if !reflect.DeepEqual(recorder.labels, want) {
					t.Fatalf("labels = %#v, want %#v", recorder.labels, want)
				}
				if !reflect.DeepEqual(recorder.expectedCID, []string{"current"}) {
					t.Fatalf("expected CID = %v", recorder.expectedCID)
				}
				if withText {
					if recorder.title == nil || *recorder.title != "New title" || recorder.body == nil || *recorder.body != "" {
						t.Fatal("explicit title and empty body not preserved")
					}
				} else if recorder.title != nil || recorder.body != nil {
					t.Fatal("label-only edit changes title or body")
				}
			})
		}
	}
}

func TestListLabelFlagsPreserveExactNamesAndOtherFilters(t *testing.T) {
	args := []string{"--label", "bug, urgent", "--label", "Help wanted", "--author", "alice.test", "--state", "open", "--limit", "3", "--order", "asc"}
	for _, command := range []*cobra.Command{newIssueListCommand(nil), newPRListCommand(nil)} {
		if err := command.ParseFlags(args); err != nil {
			t.Fatal(err)
		}
		labels, err := command.Flags().GetStringArray("label")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(labels, []string{"bug, urgent", "Help wanted"}) {
			t.Fatalf("labels = %q", labels)
		}
	}
	var flags listFlags
	command := &cobra.Command{}
	flags.add(command, "open")
	if err := command.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	options, err := flags.options(command)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(options.Labels, []string{"bug, urgent", "Help wanted"}) || options.Author != "alice.test" || options.State != "open" || options.Limit != 3 || options.Order != "asc" {
		t.Fatalf("options = %+v", options)
	}
}

func TestRenderListLabelNames(t *testing.T) {
	var output bytes.Buffer
	renderList(&output, []app.Item{{Rkey: "ticket", Title: "Problem", Labels: []app.Label{{Name: "bug"}, {Name: "help wanted"}, {URI: "at://repo/label/unknown"}}}}, "empty")
	for _, want := range []string{"LABELS", "bug, help wanted, at://repo/label/unknown"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output %q does not contain %q", output.String(), want)
		}
	}
}
