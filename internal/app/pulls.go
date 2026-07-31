package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
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

// ListPulls lists every pull request in the target repository.
func (s *Service) ListPulls(ctx context.Context, t Target) ([]Item, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, pulls.Items, decodePull), nil
}

// ViewPull finds a single pull request by rkey within the target repository.
func (s *Service) ViewPull(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %s: %w", t, err)
	}
	found, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	decoded, err := decodePull(found.Value)
	if err != nil {
		return nil, fmt.Errorf("decode pull request %q: %w", rkey, err)
	}
	return &ViewResult{
		Rkey:         rkey,
		Title:        decoded.Title,
		State:        found.State,
		Body:         decoded.Body,
		Author:       s.resolveAuthor(ctx, extractDID(found.URI)),
		CreatedAt:    decoded.CreatedAt,
		SourceBranch: decoded.SourceBranch,
		TargetBranch: decoded.TargetBranch,
	}, nil
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

func (s *Service) UpdatePullRound(ctx context.Context, repoDir, rkey string) error {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return err
	}

	return updateRecord(ctx, atClient, did, pullCollection, rkey, func(value any) (tangledlex.RepoPull, error) {
		data, err := json.Marshal(value)
		if err != nil {
			return tangledlex.RepoPull{}, fmt.Errorf("encode existing pull request: %w", err)
		}
		var record tangledlex.RepoPull
		if err := json.Unmarshal(data, &record); err != nil {
			return tangledlex.RepoPull{}, fmt.Errorf("decode existing pull request: %w", err)
		}
		if record.Target == nil || record.Source == nil || record.Target.Branch == "" || record.Source.Branch == "" {
			return tangledlex.RepoPull{}, fmt.Errorf("pull request %q has no source and target branches", rkey)
		}

		patch, err := s.git.GeneratePatch(ctx, repoDir, record.Target.Branch, record.Source.Branch)
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

// CommentPull adds a comment to the pull request identified by rkey.
func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %s: %w", t, err)
	}
	pull, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	return s.createPullComment(ctx, pull.URI, body)
}

func (s *Service) createPullComment(ctx context.Context, pullURI, body string) (*CreatedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: pullCollection + ".comment",
		Rkey:       rkey,
		Record: tangledlex.RepoPullComment{
			LexiconTypeID: pullCollection + ".comment",
			Pull:          pullURI,
			Body:          body,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create pull request comment: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
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
	pulls, err := s.appview.ListPulls(ctx, repoDID, tangled.ListOpts{
		Limit: defaultListLimit,
	})
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
	return &PullPatch{
		URI:          pull.URI,
		Title:        record.Title,
		Body:         stringValue(record.Body),
		TargetBranch: pullTargetBranch(record.Target),
		Patch:        patch,
	}, nil
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

// SetPullState closes or reopens a pull request. status is the bare verb
// ("open" or "closed").
func (s *Service) SetPullState(ctx context.Context, t Target, rkey, status string) (*StateResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	target, _, err := s.targetRecord(ctx, t, pullCollection, rkey)
	if err != nil {
		return nil, err
	}
	if err := putState(ctx, atClient, did, rkey, pullCollection, target, status); err != nil {
		return nil, err
	}
	return &StateResult{Rkey: rkey, State: status}, nil
}

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

// EditPull patches a pull request's title and/or body. A nil pointer leaves
// the field untouched.
func (s *Service) EditPull(ctx context.Context, rkey string, title, body *string) error {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return err
	}
	return editRecord(ctx, atClient, did, pullCollection, rkey, title, body)
}

// repoKnot resolves the knot host for a repository record URI.
func (s *Service) repoKnot(ctx context.Context, repoURI string) (string, error) {
	repo, err := s.appview.GetRepo(ctx, repoURI)
	if err != nil {
		return "", fmt.Errorf("get repository: %w", err)
	}
	if repo.Value.Knot == "" {
		return "", fmt.Errorf("repository record has no knot")
	}
	return repo.Value.Knot, nil
}
