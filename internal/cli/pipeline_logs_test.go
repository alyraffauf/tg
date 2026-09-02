package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

type recordingPipelineLogsService struct {
	target    app.Target
	pipeline  string
	workflows []string
	events    []app.PipelineLogEvent
	err       error
}

func (service *recordingPipelineLogsService) TargetFromCWD(context.Context) (app.Target, error) {
	return app.Target{}, errors.New("unexpected target detection")
}

func (service *recordingPipelineLogsService) PipelineStatus(context.Context, app.Target) (*app.PipelineStatusResult, error) {
	return nil, errors.New("unexpected pipeline status lookup")
}

func (service *recordingPipelineLogsService) PipelineLogs(_ context.Context, target app.Target, pipeline string, workflows []string, callback func(app.PipelineLogEvent) error) error {
	service.target = target
	service.pipeline = pipeline
	service.workflows = append([]string(nil), workflows...)
	for _, event := range service.events {
		if err := callback(event); err != nil {
			return err
		}
	}
	return service.err
}

func TestPipelineLogsJSONStreamsNDJSONToStdout(t *testing.T) {
	service := &recordingPipelineLogsService{events: []app.PipelineLogEvent{
		{Type: "data", Data: &app.PipelineLogData{Stream: "stdout", Content: "out", Workflow: "build"}},
		{Type: "data", Data: &app.PipelineLogData{Stream: "stderr", Content: "err", Workflow: "build"}},
	}}
	command := newPipelineLogsCommand(service)
	command.Flags().Bool("json", false, "")
	command.SetArgs([]string{"pipeline-1", "--repo", "owner.test/example", "--workflow", "build", "--json"})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"content":"out"`) || !strings.Contains(lines[1], `"content":"err"`) {
		t.Fatalf("NDJSON output = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if service.target.String() != "owner.test/example" || service.pipeline != "pipeline-1" || len(service.workflows) != 1 || service.workflows[0] != "build" {
		t.Fatalf("service input: target=%s pipeline=%q workflows=%q", service.target, service.pipeline, service.workflows)
	}
}

func TestPipelineLogsJSONPropagatesStreamAndWriterErrors(t *testing.T) {
	streamFailure := errors.New("stream failed")
	service := &recordingPipelineLogsService{err: streamFailure}
	command := newPipelineLogsCommand(service)
	command.Flags().Bool("json", false, "")
	command.SetArgs([]string{"pipeline-1", "--repo", "owner.test/example", "--json"})
	command.SetOut(&bytes.Buffer{})
	if err := command.Execute(); !errors.Is(err, streamFailure) {
		t.Fatalf("stream error = %v", err)
	}

	service.err = nil
	service.events = []app.PipelineLogEvent{{Type: "data", Data: &app.PipelineLogData{Content: "line"}}}
	command = newPipelineLogsCommand(service)
	command.Flags().Bool("json", false, "")
	command.SetArgs([]string{"pipeline-1", "--repo", "owner.test/example", "--json"})
	command.SetOut(errorWriter{})
	if err := command.Execute(); !errors.Is(err, errWrite) {
		t.Fatalf("writer error = %v", err)
	}
}
