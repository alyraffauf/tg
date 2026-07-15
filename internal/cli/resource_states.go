package cli

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
)

const (
	issueCollection = "sh.tangled.repo.issue"
	pullCollection  = "sh.tangled.repo.pull"
)

type issueStateRecord struct {
	Type  string `json:"$type"`
	Issue string `json:"issue"`
	State string `json:"state"`
}

type pullStatusRecord struct {
	Type   string `json:"$type"`
	Pull   string `json:"pull"`
	Status string `json:"status"`
}

type stateResult struct {
	Rkey  string `json:"rkey"`
	State string `json:"state"`
}

func targetRecord(ctx context.Context, repoArg, collection, rkey string) (string, string, error) {
	targetArgs := []string{}
	if repoArg != "" {
		targetArgs = []string{repoArg}
	}
	handle, repo, err := resolveTarget(ctx, targetArgs)
	if err != nil {
		return "", "", err
	}
	repoRecord, err := resolveRepoRecord(ctx, handle, repo)
	if err != nil {
		return "", "", err
	}

	var items []tangled.ListItem
	var recordType string
	if collection == issueCollection {
		issues, err := client.ListIssues(ctx, repoRecord.Value.RepoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return "", "", fmt.Errorf("list issues for %s/%s: %w", handle, repo, err)
		}
		items = issues.Items
		recordType = "issue"
	} else {
		pulls, err := client.ListPulls(ctx, repoRecord.Value.RepoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return "", "", fmt.Errorf("list pull requests for %s/%s: %w", handle, repo, err)
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

func putState(ctx context.Context, atClient *atproto.ATProto, did, rkey, collection, target, state string) error {
	if collection == issueCollection {
		state = "sh.tangled.repo.issue.state." + state
		return putRecord(ctx, atClient, did, "sh.tangled.repo.issue.state", rkey, issueStateRecord{
			Type:  "sh.tangled.repo.issue.state",
			Issue: target,
			State: state,
		})
	}
	state = "sh.tangled.repo.pull.status." + state
	return putRecord(ctx, atClient, did, "sh.tangled.repo.pull.status", rkey, pullStatusRecord{
		Type:   "sh.tangled.repo.pull.status",
		Pull:   target,
		Status: state,
	})
}
