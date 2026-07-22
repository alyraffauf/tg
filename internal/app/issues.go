package app

import (
	"context"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ListIssues lists every issue in the target repository.
func (s *Service) ListIssues(ctx context.Context, t Target) ([]Item, error) {
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	issues, err := s.Appview.ListIssues(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, issues.Items, decodeIssue), nil
}

// ViewIssue finds a single issue by rkey within the target repository.
func (s *Service) ViewIssue(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	issues, err := s.Appview.ListIssues(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %s: %w", t, err)
	}
	found, err := findByRKey(issues.Items, rkey, "issue")
	if err != nil {
		return nil, err
	}
	decoded, err := decodeIssue(found.Value)
	if err != nil {
		return nil, fmt.Errorf("decode issue %q: %w", rkey, err)
	}
	return &ViewResult{
		Rkey:      rkey,
		Title:     decoded.Title,
		Body:      decoded.Body,
		Author:    s.resolveAuthor(ctx, extractDID(found.URI)),
		CreatedAt: decoded.CreatedAt,
	}, nil
}

// CreateIssue writes a new issue record in the target repository.
func (s *Service) CreateIssue(ctx context.Context, t Target, title, body string) (*CreatedRecordResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: issueCollection,
		Rkey:       rkey,
		Record: tangled.IssueRecord{
			Type:      issueCollection,
			Repo:      repoDid,
			Title:     title,
			Body:      body,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}

// CommentIssue adds a comment to the issue identified by rkey.
func (s *Service) CommentIssue(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repoDid, err := s.RepoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	issues, err := s.Appview.ListIssues(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %s: %w", t, err)
	}
	issue, err := findByRKey(issues.Items, rkey, "issue")
	if err != nil {
		return nil, err
	}
	return s.createIssueComment(ctx, issue.URI, body)
}

func (s *Service) createIssueComment(ctx context.Context, issueURI, body string) (*CreatedRecordResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: issueCollection + ".comment",
		Rkey:       rkey,
		Record: tangled.IssueCommentRecord{
			Type:      issueCollection + ".comment",
			Issue:     issueURI,
			Body:      body,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create issue comment: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}

// SetIssueState closes or reopens an issue. state is the bare verb
// ("open" or "closed").
func (s *Service) SetIssueState(ctx context.Context, t Target, rkey, state string) (*StateResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	target, _, err := s.targetRecord(ctx, t, issueCollection, rkey)
	if err != nil {
		return nil, err
	}
	if err := putState(ctx, atClient, did, rkey, issueCollection, target, state); err != nil {
		return nil, err
	}
	return &StateResult{Rkey: rkey, State: state}, nil
}

// EditIssue patches an issue's title and/or body. A nil pointer leaves the
// field untouched.
func (s *Service) EditIssue(ctx context.Context, rkey string, title, body *string) error {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return err
	}
	return editRecord(ctx, atClient, did, issueCollection, rkey, title, body)
}

// targetRecord resolves t, finds the issue/pull record rkey, and returns the
// record URI and the repo record URI. collection selects issues or pulls.
func (s *Service) targetRecord(ctx context.Context, t Target, collection, rkey string) (string, string, error) {
	repoRecord, err := s.ResolveRepo(ctx, t)
	if err != nil {
		return "", "", err
	}

	var items []tangled.ListItem
	var recordType string
	if collection == issueCollection {
		issues, err := s.Appview.ListIssues(ctx, repoRecord.Value.RepoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return "", "", fmt.Errorf("list issues for %s: %w", t, err)
		}
		items = issues.Items
		recordType = "issue"
	} else {
		pulls, err := s.Appview.ListPulls(ctx, repoRecord.Value.RepoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return "", "", fmt.Errorf("list pull requests for %s: %w", t, err)
		}
		items = pulls.Items
		recordType = "pull request"
	}

	record, err := findByRKey(items, rkey, recordType)
	if err != nil {
		return "", "", err
	}
	return record.URI, repoRecord.URI, nil
}
