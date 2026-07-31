package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
)

// maxPullPatchSize caps a downloaded pull-request patch.
const maxPullPatchSize = 100 << 20

// PullPatch contains the latest decompressed patch and its target branch.
type PullPatch struct {
	URI          string
	Title        string
	Body         string
	TargetBranch string
	Patch        []byte
}

// PullPatch fetches a pull request's latest patch, decompressed and ready to
// apply or stream.
func (s *Service) PullPatch(ctx context.Context, t Target, rkey string) (*PullPatch, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	return s.pullPatch(ctx, repo, rkey)
}

func (s *Service) pullPatch(ctx context.Context, repo *tangled.Repo, rkey string) (*PullPatch, error) {
	repoDID := stringValue(repo.Value.RepoDid)
	if repoDID == "" {
		return nil, fmt.Errorf("repository has no repository DID")
	}
	pulls, err := s.appview.ListPulls(ctx, repoDID, tangled.ListOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, fmt.Errorf("list PRs for repository %q: %w", repoDID, err)
	}
	pull, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	record, patchCID, err := latestPullPatch(pull, rkey)
	if err != nil {
		return nil, err
	}
	if record.Target == nil || record.Target.Branch == "" {
		return nil, fmt.Errorf("pull request %q has no target branch", rkey)
	}
	patch, err := s.downloadPullPatch(ctx, extractDID(pull.URI), patchCID)
	if err != nil {
		return nil, err
	}
	return &PullPatch{URI: pull.URI, Title: record.Title, Body: stringValue(record.Body), TargetBranch: pullTargetBranch(record.Target), Patch: patch}, nil
}

func latestPullPatch(pull *tangled.ListItem, rkey string) (tangledlex.RepoPull, string, error) {
	var record tangledlex.RepoPull
	if err := json.Unmarshal(pull.Value, &record); err != nil {
		return record, "", fmt.Errorf("decode pull request %q: %w", rkey, err)
	}
	if len(record.Rounds) == 0 {
		return record, "", fmt.Errorf("pull request %q has no rounds", rkey)
	}
	lastRound := record.Rounds[len(record.Rounds)-1]
	if lastRound == nil || lastRound.PatchBlob == nil {
		return record, "", fmt.Errorf("pull request %q has no patch blob", rkey)
	}
	patchCID := lastRound.PatchBlob.Ref.String()
	if patchCID == "" {
		return record, "", fmt.Errorf("pull request %q has no patch blob", rkey)
	}
	return record, patchCID, nil
}

func (s *Service) downloadPullPatch(ctx context.Context, authorDID, cid string) ([]byte, error) {
	pdsHost, err := s.resolver.ResolvePDS(ctx, authorDID)
	if err != nil {
		return nil, fmt.Errorf("resolve PDS for author %q: %w", authorDID, err)
	}
	compressed, err := atproto.NewPublic(pdsHost, s.httpClient).GetBlob(ctx, authorDID, cid)
	if err != nil {
		return nil, fmt.Errorf("download patch: %w", err)
	}
	patch, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("decompress patch: %w", err)
	}
	defer patch.Close()
	contents, err := readLimited(patch, maxPullPatchSize)
	if err != nil {
		return nil, fmt.Errorf("decompress patch: %w", err)
	}
	return contents, nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(contents)) > limit {
		return nil, fmt.Errorf("patch exceeds %d bytes", limit)
	}
	return contents, nil
}
