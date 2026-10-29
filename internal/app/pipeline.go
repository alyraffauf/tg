package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/alyraffauf/tg/spindle"
)

const maxPipelinePages = 100

func (s *Service) pipelineClient(ctx context.Context, target Target) (pipelineClient, string, error) {
	spindleHost, repoDID, err := s.pipelineTarget(ctx, target)
	if err != nil {
		return nil, "", err
	}
	client, err := s.spindle.New(spindleHost)
	if err != nil {
		return nil, "", fmt.Errorf("connect to pipeline spindle: %w", err)
	}
	return client, repoDID, nil
}

func fetchOwnedPipeline(ctx context.Context, client pipelineClient, target Target, repoDID, pipelineID string) (*spindle.Pipeline, error) {
	pipeline, err := client.GetPipeline(ctx, pipelineID, repoDID)
	if err != nil {
		return nil, err
	}
	if pipeline == nil {
		return nil, fmt.Errorf("pipeline %q returned an empty response", pipelineID)
	}
	if pipeline.Repo == "" || pipeline.Repo != repoDID {
		return nil, fmt.Errorf("pipeline %q does not belong to repository %q", pipelineID, target.String())
	}
	return pipeline, nil
}

func (s *Service) pipelineTarget(ctx context.Context, target Target) (string, string, error) {
	repo, err := s.resolveRepo(ctx, target)
	if err != nil {
		return "", "", err
	}
	spindleHost := stringValue(repo.Value.Spindle)
	if strings.HasPrefix(spindleHost, "did:") {
		ident, err := s.resolver.ResolveDID(ctx, spindleHost)
		if err != nil {
			return "", "", fmt.Errorf("resolve spindle DID: %w", err)
		}
		service := ident.Services["tangled_spindle"]
		if service.Type != "TangledSpindle" || service.URL == "" {
			return "", "", fmt.Errorf("spindle DID has no TangledSpindle service")
		}
		spindleHost = service.URL
	}

	if spindleHost == "" {
		return "", "", fmt.Errorf("pipelines are not configured for repository %q", target.String())
	}
	repoDID := stringValue(repo.Value.RepoDid)
	if repoDID == "" {
		return "", "", fmt.Errorf("repository %q has no repository DID", target.String())
	}
	return spindleHost, repoDID, nil
}

func pipelineItems(pipelines []spindle.Pipeline) []Pipeline {
	items := make([]Pipeline, 0, len(pipelines))
	for _, pipeline := range pipelines {
		items = append(items, pipelineItem(pipeline))
	}
	return items
}

func pipelineItem(pipeline spindle.Pipeline) Pipeline {
	workflows := make([]PipelineWorkflow, 0, len(pipeline.Workflows))
	for _, workflow := range pipeline.Workflows {
		workflows = append(workflows, PipelineWorkflow{
			ID: workflow.ID, Name: workflow.Name, Status: workflow.Status, Error: workflow.Error,
			StartedAt: workflow.StartedAt, FinishedAt: workflow.FinishedAt,
		})
	}
	return Pipeline{
		ID: pipeline.ID, Commit: pipeline.Commit, CreatedAt: pipeline.CreatedAt,
		Repo: pipeline.Repo, SourceRepo: pipeline.SourceRepo, Trigger: pipeline.Trigger, Workflows: workflows,
	}
}

func (s *Service) spindleToken(ctx context.Context, target Target, pds pdsClient, host, repoDID, method string) (string, error) {
	repo, err := s.resolveVerifiedRepo(ctx, target)
	if err != nil {
		return "", err
	}
	audience := stringValue(repo.Value.Spindle)
	if !strings.HasPrefix(audience, "did:") {
		audience, err = spindle.ServiceDID(host)
		if err != nil {
			return "", err
		}
	}
	capabilities, err := s.knot.NewPublic(repo.Value.Knot).DescribeKnot(ctx)
	if err != nil {
		return "", err
	}
	if slices.Contains(capabilities.Capabilities, "knot-repo-token-mint") {
		token, err := pds.GetServiceAuth(ctx, repoDID, "org.tangled.repo.mintToken")
		if err != nil {
			return "", err
		}
		var out struct {
			Token string `json:"token"`
		}
		if err := s.knot.New(repo.Value.Knot, token).Procedure(ctx, "org.tangled.repo.mintToken", map[string]any{"repo": repoDID, "aud": audience, "lxm": method, "exp": time.Now().Unix() + 55}, &out); err != nil {
			return "", err
		}
		if out.Token == "" {
			return "", fmt.Errorf("knot returned an empty repository token")
		}
		return out.Token, nil
	}
	return pds.GetServiceAuth(ctx, audience, method)
}
