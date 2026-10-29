//go:build upstream_integration

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Pipeline checks extend the disposable PR fixture when a real Spindle is running.
func configureUpstreamPipeline(t *testing.T, service *Service, ctx context.Context, target Target, dir string) {
	t.Helper()
	spindle := "did:web:spindle.tngl.boltless.dev"
	edited, err := service.EditRepo(ctx, target, EditRepoInput{Spindle: &spindle})
	if err != nil || len(edited.Warnings) != 0 {
		t.Fatalf("configure spindle=%+v err=%v", edited, err)
	}
	path := filepath.Join(dir, ".tangled", "workflows", "ci.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	workflow := "engine: dummy\nwhen:\n  - event: [push, pull_request]\n    branch: [main]\nsteps:\n  - name: verify\n    command: echo verified\n"
	if err := os.WriteFile(path, []byte(workflow), 0600); err != nil {
		t.Fatal(err)
	}
	stackGit(t, ctx, dir, "add", ".tangled/workflows/ci.yml")
}

func awaitUpstreamPipeline(t *testing.T, service *Service, ctx context.Context, target Target, id string) *Pipeline {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		pipeline, err := service.ViewPipeline(ctx, target, id)
		if err != nil {
			t.Fatal(err)
		}
		complete := len(pipeline.Workflows) > 0
		for _, workflow := range pipeline.Workflows {
			switch workflow.Status {
			case "success":
			case "pending", "running":
				complete = false
			default:
				t.Fatalf("fixture workflow failed: %+v", workflow)
			}
		}
		if complete {
			return pipeline
		}
		if time.Now().After(deadline) {
			t.Fatalf("fixture pipeline did not finish: %+v", pipeline)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func verifyUpstreamPipeline(t *testing.T, service *Service, ctx context.Context, target Target, dir string) {
	t.Helper()
	commit := stackGit(t, ctx, dir, "rev-parse", "main")
	triggered, err := service.TriggerPipeline(ctx, target, commit, nil)
	if err != nil || len(triggered.Pipelines) != 1 || triggered.Pipeline != triggered.Pipelines[0] {
		t.Fatalf("trigger=%+v err=%v", triggered, err)
	}
	pipeline := awaitUpstreamPipeline(t, service, ctx, target, triggered.Pipeline)
	if pipeline.Commit != commit || pipeline.Trigger["$type"] != "org.tangled.ci.trigger.manual" {
		t.Fatalf("manual pipeline=%+v", pipeline)
	}
	logContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var logs strings.Builder
	sawOutcome := false
	err = service.PipelineLogs(logContext, target, pipeline.ID, nil, func(event PipelineLogEvent) error {
		if event.Control != nil && event.Control.Status == "end" && event.Control.Outcome == "success" {
			sawOutcome = true
		}
		if event.Data != nil {
			logs.WriteString(event.Data.Content)
		}
		return nil
	})
	if !strings.Contains(logs.String(), "echo verified") || !sawOutcome {
		t.Fatalf("logs=%q err=%v", logs.String(), err)
	}
	if err != nil {
		var closeError *websocket.CloseError
		if !errors.As(err, &closeError) || closeError.Code != websocket.CloseAbnormalClosure {
			t.Fatal(err)
		}
		// Upstream drains its writer then calls Close without a close frame or
		// terminal event. Keep this defect visible instead of declaring delivery complete.
		t.Logf("upstream log completion remains unconfirmed: %v", err)
	}
	retried, err := service.RetryPipeline(ctx, target, pipeline.ID, nil)
	if err != nil || retried.Pipeline == pipeline.ID || retried.OriginalPipeline != pipeline.ID {
		t.Fatalf("retry=%+v err=%v", retried, err)
	}
	awaitUpstreamPipeline(t, service, ctx, target, retried.Pipeline)
	listed, err := service.ListPipelines(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listed {
		if item.ID == retried.Pipeline {
			found = true
		}
	}
	if !found {
		t.Fatal("retried pipeline absent from listing")
	}
	cancelled, err := service.CancelPipeline(ctx, target, retried.Pipeline, nil)
	if err != nil || cancelled.CancellationRequested {
		t.Fatalf("finished cancellation=%+v err=%v", cancelled, err)
	}
	t.Logf("verified manual pipeline %s, retry %s, and archived log retrieval", pipeline.ID, retried.Pipeline)
}

func verifyUpstreamPullPipeline(t *testing.T, service *Service, ctx context.Context, target Target, pullURI, head string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		pipelines, err := service.ListPipelines(ctx, target)
		if err != nil {
			t.Fatal(err)
		}
		for _, pipeline := range pipelines {
			if pipeline.Trigger["pull"] != pullURI || pipeline.Trigger["sourceSha"] != head {
				continue
			}
			if pipeline.Commit != head || pipeline.Trigger["sourceBranch"] != "proposal" || pipeline.Trigger["targetBranch"] != "main" || pipeline.Trigger["action"] == "" {
				t.Fatalf("pull pipeline lost its association: %+v", pipeline)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pull %s has no associated pipeline for %s", pullURI, head)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}
