package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
)

type CreatePullInput struct {
	RepoDir string
	Title   string
	Body    string
	Base    string
	Head    string
	Target  Target
	Source  *Target
}

var errForkPullBaseRequired = errors.New("fork pull requests require --base; pass the target remote branch, for example --base upstream/main")

// PullBaseUnavailableError means the published target tip is absent locally.
type PullBaseUnavailableError struct {
	Branch          string
	LocalCommit     string
	PublishedCommit string
	Err             error
}

func (err *PullBaseUnavailableError) Error() string {
	return fmt.Sprintf("target branch %s is published at %s, but local base %s cannot provide that commit; fetch the target repository and retry: %v", err.Branch, err.PublishedCommit, err.LocalCommit, err.Err)
}

func (err *PullBaseUnavailableError) Unwrap() error { return err.Err }

type pullRange struct {
	Base         string
	Head         string
	Branch       string
	SourceBranch string
	SourceRepo   string
}

func (s *Service) resolvePullRange(ctx context.Context, in CreatePullInput, target *tangled.Repo) (pullRange, error) {
	source := target
	var err error
	if in.Source != nil {
		source, err = s.resolveRepo(ctx, *in.Source)
		if err != nil {
			return pullRange{}, err
		}
	}
	head := in.Head
	if head == "" {
		head, err = s.git.CurrentBranch(ctx, in.RepoDir)
		if err != nil {
			return pullRange{}, err
		}
	}
	headOID, err := s.git.ResolveCommit(ctx, in.RepoDir, head)
	if err != nil {
		return pullRange{}, err
	}
	var base gitutil.PullBase
	if in.Base != "" {
		base, err = s.git.ResolvePullBase(ctx, in.RepoDir, in.Base)
	} else {
		if stringValue(source.Value.RepoDid) != stringValue(target.Value.RepoDid) {
			return pullRange{}, errForkPullBaseRequired
		}
		base, err = s.git.DefaultPullBase(ctx, in.RepoDir)
	}
	if err != nil {
		return pullRange{}, err
	}
	targetTip, err := s.appview.GetRef(ctx, stringValue(target.Value.RepoDid), "refs/heads/"+base.Branch)
	if err != nil {
		return pullRange{}, fmt.Errorf("confirm published target branch %s: %w", base.Branch, explainMirrorError(err))
	}
	if targetTip != base.Revision {
		if _, err := s.git.ResolveCommit(ctx, in.RepoDir, targetTip); err != nil {
			return pullRange{}, &PullBaseUnavailableError{Branch: base.Branch, LocalCommit: base.Revision, PublishedCommit: targetTip, Err: err}
		}
	}
	mergeBase, err := s.git.MergeBase(ctx, in.RepoDir, targetTip, headOID)
	if err != nil {
		return pullRange{}, err
	}
	if mergeBase == headOID {
		return pullRange{}, fmt.Errorf("no commits to propose")
	}
	sourceBranch := strings.TrimPrefix(head, "refs/heads/")
	remoteHead, err := s.appview.GetRef(ctx, stringValue(source.Value.RepoDid), "refs/heads/"+sourceBranch)
	if err != nil {
		return pullRange{}, fmt.Errorf("confirm published source branch %s: %w; retry once the source repository is available in the Git mirror", sourceBranch, err)
	}
	if remoteHead != headOID {
		return pullRange{}, fmt.Errorf("source commit %s is not the published tip of %s; push that branch to the source Tangled repository and wait for the Git mirror before submitting the PR", headOID, sourceBranch)
	}
	change := pullRange{Base: mergeBase, Head: headOID, Branch: base.Branch, SourceBranch: sourceBranch, SourceRepo: stringValue(source.Value.RepoDid)}
	if _, err := s.formatPullPatch(ctx, change); err != nil {
		return pullRange{}, err
	}
	return change, nil
}
func (s *Service) submitPullPatch(ctx context.Context, repo *tangled.Repo, ticket string, change pullRange) (*knot.Receipt, []string, error) {
	patch := map[string]any{"ticket": extractRKey(ticket), "base": map[string]string{"oid": change.Base}, "head": map[string]string{"oid": change.Head}, "sourceRepo": change.SourceRepo, "sourceBranch": change.SourceBranch}
	receipt, warnings, err := s.socialProcedure(ctx, repo, "org.tangled.review.submitPatch", map[string]any{"patch": patch})
	return receipt, warnings, err
}
func (s *Service) CreatePull(ctx context.Context, in CreatePullInput) (*PRCreateResult, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("title must not be empty")
	}
	repo, err := s.resolveRepo(ctx, in.Target)
	if err != nil {
		return nil, err
	}
	if err := s.requireNative(ctx, repo); err != nil {
		return nil, err
	}
	change, err := s.resolvePullRange(ctx, in, repo)
	if err != nil {
		return nil, err
	}
	receipt, warnings, err := s.socialProcedure(ctx, repo, "com.atproto.repo.createRecord", map[string]any{"collection": tangled.IssueCollection, "record": ticketRecord(in.Title, in.Body, change.Branch)})
	if err != nil {
		return nil, err
	}
	result := &PRCreateResult{PatchStatus: PatchSubmitted, URI: receipt.URI, CID: receipt.CID, Title: in.Title, Base: change.Branch, Head: change.SourceBranch, Warnings: warnings}
	_, patchWarnings, err := s.submitPullPatch(ctx, repo, receipt.URI, change)
	if err == nil {
		result.Warnings = append(result.Warnings, patchWarnings...)
		return result, nil
	}
	// A transport failure may have saved the patch. Only erase after a definite rejection.
	if !isDefiniteRejection(err) {
		result.PatchStatus = PatchUnknown
		return result, fmt.Errorf("submit patch for %s: %w; inspect this ticket before retrying", receipt.URI, err)
	}
	if cleanupErr := s.eraseSocial(ctx, repo, *receipt); cleanupErr != nil {
		result.PatchStatus = PatchRejected
		return result, fmt.Errorf("submit patch for %s: %w; the ticket could not be removed: %v", receipt.URI, err, cleanupErr)
	}
	return nil, fmt.Errorf("submit patch: %w; ticket %s was removed", err, receipt.URI)
}
func atURIPrefix(uri string) bool { return strings.HasPrefix(uri, "at://") }
