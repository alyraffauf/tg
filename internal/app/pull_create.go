package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
)

// CreatePullInput configures pull request creation.
type CreatePullInput struct {
	RepoDir string // local git repository (for branch detection + patch)
	Title   string
	Body    string
	Base    string // empty: detect origin's default branch
	Head    string // empty: current branch
	Target  Target
	Source  *Target // nil: same as Target
}

// pullRecordInput is the write-side input to newPullRecord.
type pullRecordInput struct {
	Title         string
	Body          string
	TargetRepoDid string
	SourceRepoDid string
	Base          string
	Head          string
	Patch         *atproto.Blob
}

// CreatePull generates a patch from the local repository, uploads it, and
// writes a pull record.
func (s *Service) CreatePull(ctx context.Context, in CreatePullInput) (*PRCreateResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}

	head := in.Head
	if head == "" {
		head, err = s.git.CurrentBranch(ctx, in.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("determine source branch: %w", err)
		}
	}
	base := in.Base
	if base == "" {
		base, err = s.git.DefaultBranch(ctx, in.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("determine target branch; set --base explicitly: %w", err)
		}
	}

	target, err := s.resolveRepo(ctx, in.Target)
	if err != nil {
		return nil, err
	}
	if !atURIPrefix(target.URI) {
		return nil, fmt.Errorf("target repository %q has no strong at:// URI", in.Target.Repo)
	}
	source := target
	if in.Source != nil {
		source, err = s.resolveRepo(ctx, *in.Source)
		if err != nil {
			return nil, fmt.Errorf("resolve source repository: %w", err)
		}
	}
	if stringValue(source.Value.RepoDid) == "" {
		return nil, fmt.Errorf("source repository has no repo DID")
	}

	patch, err := s.git.GeneratePatch(ctx, in.RepoDir, base, head)
	if err != nil {
		return nil, fmt.Errorf("generate pull request patch: %w", err)
	}
	blob, err := atClient.UploadBlob(ctx, patch, patchMimeType)
	if err != nil {
		return nil, err
	}

	uri, err := createPullRecord(ctx, atClient, did, pullRecordInput{
		Title:         in.Title,
		Body:          in.Body,
		TargetRepoDid: stringValue(target.Value.RepoDid),
		SourceRepoDid: stringValue(source.Value.RepoDid),
		Base:          base,
		Head:          head,
		Patch:         blob,
	})
	if err != nil {
		return nil, err
	}
	return &PRCreateResult{URI: uri, Title: in.Title, Base: base, Head: head}, nil
}

func atURIPrefix(uri string) bool { return len(uri) >= 5 && uri[:5] == "at://" }

func createPullRecord(ctx context.Context, atClient pdsClient, did string, input pullRecordInput) (string, error) {
	record, err := newPullRecord(input, time.Now().UTC())
	if err != nil {
		return "", err
	}
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: pullCollection,
		Rkey:       string(syntax.NewTIDNow(0)),
		Record:     record,
	})
	if err != nil {
		return "", fmt.Errorf("create pull request record: %w", err)
	}
	return uri, nil
}

func newPullRecord(input pullRecordInput, createdAt time.Time) (tangledlex.RepoPull, error) {
	now := createdAt.Format(time.RFC3339)
	patchBlob, err := patchBlob(input.Patch)
	if err != nil {
		return tangledlex.RepoPull{}, err
	}
	return tangledlex.RepoPull{
		LexiconTypeID: pullCollection,
		Title:         input.Title,
		Body:          optionalString(input.Body),
		CreatedAt:     now,
		Target: &tangledlex.RepoPull_Target{
			Repo:   input.TargetRepoDid,
			Branch: input.Base,
		},
		Source: &tangledlex.RepoPull_Source{
			Repo:   optionalString(input.SourceRepoDid),
			Branch: input.Head,
		},
		Rounds: []*tangledlex.RepoPull_Round{{
			CreatedAt: now,
			PatchBlob: &patchBlob,
		}},
	}, nil
}

func patchBlob(blob *atproto.Blob) (lexutil.LexBlob, error) {
	if blob == nil || blob.Ref == nil {
		return lexutil.LexBlob{}, nil
	}
	data, err := json.Marshal(blob)
	if err != nil {
		return lexutil.LexBlob{}, fmt.Errorf("encode pull patch blob: %w", err)
	}
	var result lexutil.LexBlob
	if err := json.Unmarshal(data, &result); err != nil {
		return lexutil.LexBlob{}, fmt.Errorf("decode pull patch blob: %w", err)
	}
	return result, nil
}
