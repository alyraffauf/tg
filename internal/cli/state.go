package cli

import (
	"context"
	"encoding/json"
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

func authenticatedATProto(ctx context.Context) (*atproto.ATProto, string, error) {
	if auth == nil || !auth.IsAuthenticated() {
		return nil, "", fmt.Errorf("not logged in; run \"tg auth login\" first")
	}

	pds, err := auth.APIClient(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get auth client: %w", err)
	}
	return &atproto.ATProto{Client: pds}, auth.CurrentDID().String(), nil
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

func putRecord(ctx context.Context, atClient *atproto.ATProto, did, collection, rkey string, record any) error {
	if _, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo: did, Collection: collection, Rkey: rkey, Record: record,
	}); err != nil {
		return err
	}
	return nil
}

func editRecord(ctx context.Context, atClient *atproto.ATProto, did, collection, rkey, title, body string, setTitle, setBody bool) error {
	found, err := atClient.GetRecord(ctx, did, collection, rkey)
	if err != nil {
		return fmt.Errorf("get existing record: %w", err)
	}

	record, err := preserveRecord(found.Value)
	if err != nil {
		return err
	}
	if setTitle {
		record["title"] = title
	}
	if setBody {
		record["body"] = body
	}
	_, _, err = atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo: did, Collection: collection, Rkey: rkey, Record: record,
	})
	return err
}

func preserveRecord(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode existing record: %w", err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decode existing record: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("existing record is not an object")
	}
	return record, nil
}
