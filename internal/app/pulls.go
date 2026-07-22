package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// maxPullPatchSize caps a downloaded pull-request patch.
const maxPullPatchSize = 100 << 20

// PullPatch is a pull request's decoded record and its latest decompressed
// patch, returned by PullPatch for use by diff and checkout.
type PullPatch struct {
	URI    string
	Record tangled.PullRecord
	Patch  []byte
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
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.Appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, pulls.Items, decodePull), nil
}

// ViewPull finds a single pull request by rkey within the target repository.
func (s *Service) ViewPull(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.Appview.ListPulls(ctx, repoDid, tangled.ListOpts{
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
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}

	head := in.Head
	if head == "" {
		head, err = s.Git.CurrentBranch(ctx, in.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("determine source branch: %w", err)
		}
	}
	base := in.Base
	if base == "" {
		base, err = s.Git.DefaultBranch(ctx, in.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("determine target branch; set --base explicitly: %w", err)
		}
	}

	target, err := s.ResolveRepo(ctx, in.Target)
	if err != nil {
		return nil, err
	}
	if !atURIPrefix(target.URI) {
		return nil, fmt.Errorf("target repository %q has no strong at:// URI", in.Target.Repo)
	}
	source := target
	if in.Source != nil {
		source, err = s.ResolveRepo(ctx, *in.Source)
		if err != nil {
			return nil, fmt.Errorf("resolve source repository: %w", err)
		}
	}
	if source.Value.RepoDid == "" {
		return nil, fmt.Errorf("source repository has no repo DID")
	}

	patch, err := s.Git.GeneratePatch(ctx, in.RepoDir, base, head)
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
		TargetRepoDid: target.Value.RepoDid,
		SourceRepoDid: source.Value.RepoDid,
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

func createPullRecord(ctx context.Context, atClient *atproto.ATProto, did string, input pullRecordInput) (string, error) {
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

func newPullRecord(input pullRecordInput, createdAt time.Time) (tangled.PullRecord, error) {
	now := createdAt.Format(time.RFC3339)
	patchBlob, err := patchBlob(input.Patch)
	if err != nil {
		return tangled.PullRecord{}, err
	}
	return tangled.PullRecord{
		Type:      pullCollection,
		Title:     input.Title,
		Body:      input.Body,
		CreatedAt: now,
		Target: tangled.PullTarget{
			Repo:   input.TargetRepoDid,
			Branch: input.Base,
		},
		Source: tangled.PullSource{
			Repo:   input.SourceRepoDid,
			Branch: input.Head,
		},
		Rounds: []tangled.PullRound{{
			CreatedAt: now,
			PatchBlob: patchBlob,
		}},
	}, nil
}

func patchBlob(blob *atproto.Blob) (tangled.PatchBlob, error) {
	if blob == nil || blob.Ref == nil {
		return tangled.PatchBlob{}, nil
	}
	data, err := json.Marshal(blob)
	if err != nil {
		return tangled.PatchBlob{}, fmt.Errorf("encode pull patch blob: %w", err)
	}
	var result tangled.PatchBlob
	if err := json.Unmarshal(data, &result); err != nil {
		return tangled.PatchBlob{}, fmt.Errorf("decode pull patch blob: %w", err)
	}
	return result, nil
}

// CommentPull adds a comment to the pull request identified by rkey.
func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.Appview.ListPulls(ctx, repoDid, tangled.ListOpts{
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
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: pullCollection + ".comment",
		Rkey:       rkey,
		Record: tangled.PullCommentRecord{
			Type:      pullCollection + ".comment",
			Pull:      pullURI,
			Body:      body,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
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
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.Appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %s: %w", t, err)
	}
	pull, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	record, patchCID, err := latestPullPatch(pull, rkey)
	if err != nil {
		return nil, err
	}
	patch, err := s.downloadPullPatch(ctx, extractDID(pull.URI), patchCID)
	if err != nil {
		return nil, err
	}
	return &PullPatch{URI: pull.URI, Record: record, Patch: patch}, nil
}

func latestPullPatch(pull *tangled.ListItem, rkey string) (tangled.PullRecord, string, error) {
	var record tangled.PullRecord
	if err := json.Unmarshal(pull.Value, &record); err != nil {
		return record, "", fmt.Errorf("decode pull request %q: %w", rkey, err)
	}
	if len(record.Rounds) == 0 {
		return record, "", fmt.Errorf("pull request %q has no rounds", rkey)
	}
	patchCID := record.Rounds[len(record.Rounds)-1].PatchBlob.Ref.String()
	if patchCID == "" {
		return record, "", fmt.Errorf("pull request %q has no patch blob", rkey)
	}
	return record, patchCID, nil
}

func (s *Service) downloadPullPatch(ctx context.Context, authorDID, cid string) ([]byte, error) {
	pdsHost, err := s.Resolver.ResolvePDS(ctx, authorDID)
	if err != nil {
		return nil, fmt.Errorf("resolve PDS for author %q: %w", authorDID, err)
	}
	url := fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s", pdsHost, authorDID, cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build patch download request: %w", err)
	}
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download patch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download patch: PDS returned HTTP %d", resp.StatusCode)
	}

	compressed, err := readLimited(resp.Body, maxPullPatchSize)
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
	atClient, did, err := s.AuthenticatedClient(ctx)
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
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	pullURI, repoURI, err := s.targetRecord(ctx, t, pullCollection, rkey)
	if err != nil {
		return nil, err
	}
	knotHost, err := s.repoKnot(ctx, repoURI)
	if err != nil {
		return nil, err
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.merge")
	if err != nil {
		return nil, err
	}
	if err := knot.New(knotHost, token).Merge(ctx, knot.MergeInput{Repo: repoURI, Pull: pullURI}); err != nil {
		return nil, err
	}
	if err := putState(ctx, atClient, did, rkey, pullCollection, pullURI, "merged"); err != nil {
		return nil, fmt.Errorf("record merged pull request status: %w", err)
	}
	return &StateResult{Rkey: rkey, State: "merged"}, nil
}

// EditPull patches a pull request's title and/or body. A nil pointer leaves
// the field untouched.
func (s *Service) EditPull(ctx context.Context, rkey string, title, body *string) error {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return err
	}
	return editRecord(ctx, atClient, did, pullCollection, rkey, title, body)
}

// repoKnot resolves the knot host for a repository record URI.
func (s *Service) repoKnot(ctx context.Context, repoURI string) (string, error) {
	repo, err := s.Appview.GetRepo(ctx, repoURI)
	if err != nil {
		return "", fmt.Errorf("get repository: %w", err)
	}
	if repo.Value.Knot == "" {
		return "", fmt.Errorf("repository record has no knot")
	}
	return repo.Value.Knot, nil
}
