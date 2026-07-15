package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type issueCommentRecord struct {
	Type      string `json:"$type"`
	Issue     string `json:"issue"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

func commandBody(body, bodyFile string) (string, error) {
	if bodyFile == "" {
		return body, nil
	}
	if body != "" {
		return "", fmt.Errorf("--body and --body-file cannot be used together")
	}
	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return "", fmt.Errorf("read body file: %w", err)
	}
	return string(data), nil
}

type pullCommentRecord struct {
	Type      string `json:"$type"`
	Pull      string `json:"pull"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type createdRecordResult struct {
	Rkey string `json:"rkey"`
	URI  string `json:"uri"`
}

func createIssueComment(ctx context.Context, issueURI, body string) (createdRecordResult, error) {
	atClient, did, err := authenticatedATProto(ctx)
	if err != nil {
		return createdRecordResult{}, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: "sh.tangled.repo.issue.comment",
		Rkey:       rkey,
		Record: issueCommentRecord{
			Type:      "sh.tangled.repo.issue.comment",
			Issue:     issueURI,
			Body:      body,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return createdRecordResult{}, fmt.Errorf("create issue comment: %w", err)
	}
	return createdRecordResult{Rkey: rkey, URI: uri}, nil
}

func createPullComment(ctx context.Context, pullURI, body string) (createdRecordResult, error) {
	atClient, did, err := authenticatedATProto(ctx)
	if err != nil {
		return createdRecordResult{}, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: "sh.tangled.repo.pull.comment",
		Rkey:       rkey,
		Record: pullCommentRecord{
			Type:      "sh.tangled.repo.pull.comment",
			Pull:      pullURI,
			Body:      body,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return createdRecordResult{}, fmt.Errorf("create pull request comment: %w", err)
	}
	return createdRecordResult{Rkey: rkey, URI: uri}, nil
}
