package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/knot"
)

// MergePull applies a pull request on its knot and records the merged status.
func (s *Service) MergePull(ctx context.Context, t Target, rkey string) (*StateResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	pull, err := s.pullPatch(ctx, repo, rkey)
	if err != nil {
		return nil, err
	}
	repoDID := stringValue(repo.Value.RepoDid)
	if repoDID == "" {
		return nil, fmt.Errorf("repository %q has no repository DID", t.String())
	}
	repoName := stringValue(repo.Value.Name)
	if repoName == "" {
		repoName = t.Repo
	}
	knotHost := repo.Value.Knot
	if knotHost == "" {
		return nil, fmt.Errorf("repository %q has no knot", t.String())
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.merge")
	if err != nil {
		return nil, err
	}
	commitMessage := pull.Title
	commitBody := pull.Body
	if err := s.knot.New(knotHost, token).Merge(ctx, knot.MergeInput{
		DID: extractDID(repo.URI), Name: repoName, Repo: repoDID, Branch: pull.TargetBranch, Patch: string(pull.Patch),
		CommitMessage: &commitMessage, CommitBody: optionalString(commitBody),
	}); err != nil {
		return nil, err
	}
	if err := putState(ctx, atClient, did, rkey, pullCollection, pull.URI, "merged"); err != nil {
		return nil, fmt.Errorf("record merged pull request status: %w", err)
	}
	return &StateResult{Rkey: rkey, State: "merged"}, nil
}
