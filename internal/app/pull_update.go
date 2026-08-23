package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

// UpdatePullRoundInput configures a new round for an existing pull request.
type UpdatePullRoundInput struct {
	RepoDir string
	Rkey    string
	Base    string
}

// UpdatePullRound generates a fresh patch for an existing pull request and
// appends it as a new round, using compare-and-swap on the record CID.
func (s *Service) UpdatePullRound(ctx context.Context, in UpdatePullRoundInput) error {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return err
	}

	return updateRecord(ctx, atClient, did, pullCollection, in.Rkey, func(value any) (tangledlex.RepoPull, error) {
		data, err := json.Marshal(value)
		if err != nil {
			return tangledlex.RepoPull{}, fmt.Errorf("encode existing pull request: %w", err)
		}
		var record tangledlex.RepoPull
		if err := json.Unmarshal(data, &record); err != nil {
			return tangledlex.RepoPull{}, fmt.Errorf("decode existing pull request: %w", err)
		}
		if record.Target == nil || record.Source == nil || record.Target.Branch == "" || record.Source.Branch == "" {
			return tangledlex.RepoPull{}, fmt.Errorf("pull request %q has no source and target branches", in.Rkey)
		}

		baseInput := in.Base
		if baseInput == "" {
			sourceRepoDID := stringValue(record.Source.Repo)
			if sourceRepoDID != "" && sourceRepoDID != record.Target.Repo {
				return tangledlex.RepoPull{}, errForkPullBaseRequired
			}
			baseInput = "origin/" + record.Target.Branch
		}
		base, err := s.git.ResolvePullBase(ctx, in.RepoDir, baseInput)
		if err != nil {
			return tangledlex.RepoPull{}, fmt.Errorf("resolve target branch: %w", err)
		}
		if base.Branch != record.Target.Branch {
			return tangledlex.RepoPull{}, fmt.Errorf(
				"base %q resolves to target branch %q, but pull request targets %q",
				baseInput, base.Branch, record.Target.Branch,
			)
		}

		patch, err := s.git.GeneratePatch(ctx, in.RepoDir, base.Revision, record.Source.Branch)
		if err != nil {
			return tangledlex.RepoPull{}, fmt.Errorf("generate pull request patch: %w", err)
		}
		blob, err := atClient.UploadBlob(ctx, patch, patchMimeType)
		if err != nil {
			return tangledlex.RepoPull{}, err
		}
		patchBlob, err := patchBlob(blob)
		if err != nil {
			return tangledlex.RepoPull{}, err
		}
		record.Rounds = append(record.Rounds, &tangledlex.RepoPull_Round{
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			PatchBlob: &patchBlob,
		})
		return record, nil
	})
}
