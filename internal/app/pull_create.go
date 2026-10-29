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
	mergeBase, err := s.git.MergeBase(ctx, in.RepoDir, base.Revision, headOID)
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
	return pullRange{Base: mergeBase, Head: headOID, Branch: base.Branch, SourceBranch: sourceBranch, SourceRepo: stringValue(source.Value.RepoDid)}, nil
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
	_, patchWarnings, err := s.submitPullPatch(ctx, repo, receipt.URI, change)
	if err != nil {
		// A transport failure may have saved the patch. Only erase after a definite rejection.
		if isDefiniteRejection(err) {
			if cleanupErr := s.eraseSocial(ctx, repo, *receipt); cleanupErr != nil {
				return nil, fmt.Errorf("submit patch: %w; ticket %s could not be removed: %v", err, receipt.URI, cleanupErr)
			}
		}
		return nil, fmt.Errorf("submit patch for %s: %w; inspect this ticket before retrying", receipt.URI, err)
	}
	return &PRCreateResult{URI: receipt.URI, CID: receipt.CID, Title: in.Title, Base: change.Branch, Head: change.SourceBranch, Warnings: append(warnings, patchWarnings...)}, nil
}
func atURIPrefix(uri string) bool { return strings.HasPrefix(uri, "at://") }
