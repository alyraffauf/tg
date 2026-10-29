package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

type UpdatePullRoundInput struct {
	RepoDir string
	Rkey    string
	Base    string
	Head    string
	Target  Target
	Source  *Target
}

func (s *Service) UpdatePullRound(ctx context.Context, in UpdatePullRoundInput) (*UpdatedRecordResult, error) {
	repo, err := s.resolveRepo(ctx, in.Target)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, stringValue(repo.Value.RepoDid), tangled.ListOpts{Limit: 100})
	if err != nil {
		return nil, err
	}
	item, err := findByRKey(pulls.Items, in.Rkey, "pull request")
	if err != nil {
		return nil, err
	}
	ticket, err := decodePull(item.Value)
	if err != nil {
		return nil, err
	}
	patches, err := s.appview.ListPatches(ctx, item.URI)
	if err != nil {
		return nil, err
	}
	source := in.Source
	if source == nil && len(patches) > 0 {
		last := patches[len(patches)-1]
		if last.Value.Source != nil {
			source = &Target{RepoDID: last.Value.Source.Repo}
			if in.Head == "" {
				in.Head = last.Value.Source.Branch
			}
		}
	}
	base := in.Base
	if base == "" {
		if source != nil {
			resolved, err := s.resolveRepo(ctx, *source)
			if err != nil {
				return nil, err
			}
			if stringValue(resolved.Value.RepoDid) != stringValue(repo.Value.RepoDid) {
				return nil, errForkPullBaseRequired
			}
		}
		base = "origin/" + ticket.TargetBranch
	}
	change, err := s.resolvePullRange(ctx, CreatePullInput{RepoDir: in.RepoDir, Base: base, Head: in.Head, Source: source}, repo)
	if err != nil {
		return nil, err
	}
	if change.Branch != ticket.TargetBranch {
		return nil, fmt.Errorf("base targets %q but ticket targets %q", change.Branch, ticket.TargetBranch)
	}
	receipt, warnings, err := s.submitPullPatch(ctx, repo, item.URI, change)
	if err != nil {
		return nil, err
	}
	return &UpdatedRecordResult{Rkey: in.Rkey, URI: receipt.URI, CID: receipt.CID, Warnings: warnings}, nil
}
