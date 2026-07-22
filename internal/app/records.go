package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
)

const (
	issueCollection  = tangled.IssueCollection
	pullCollection   = tangled.PullCollection
	stringCollection = tangled.StringCollection
	sshKeyCollection = tangled.SSHKeyCollection
	repoCollection   = tangled.RepoCollection
)

const patchMimeType = "application/gzip"

// putRecord writes a record to the PDS.
func putRecord(ctx context.Context, atClient *atproto.ATProto, did, collection, rkey string, record any) error {
	if _, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo: did, Collection: collection, Rkey: rkey, Record: record,
	}); err != nil {
		return err
	}
	return nil
}

// editRecord fetches an existing record, applies the provided title and/or
// body patches (nil leaves the field untouched), and writes it back.
func editRecord(ctx context.Context, atClient *atproto.ATProto, did, collection, rkey string, title, body *string) error {
	found, err := atClient.GetRecord(ctx, did, collection, rkey)
	if err != nil {
		return fmt.Errorf("get existing record: %w", err)
	}

	record, err := preserveRecord(found.Value)
	if err != nil {
		return err
	}
	if title != nil {
		record["title"] = *title
	}
	if body != nil {
		record["body"] = *body
	}
	_, _, err = atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo: did, Collection: collection, Rkey: rkey, Record: record,
	})
	return err
}

// preserveRecord marshals a record value into a map so individual fields can
// be patched without losing fields this client does not model.
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

// putState writes an issue.state or pull.status record keyed by rkey. state
// is the bare verb ("open"/"closed"/"merged"); the collection-specific suffix
// is applied here.
func putState(ctx context.Context, atClient *atproto.ATProto, did, rkey, collection, target, state string) error {
	if collection == tangled.IssueCollection {
		state = tangled.IssueCollection + tangled.IssueStateSuffix + "." + state
		return putRecord(ctx, atClient, did, tangled.IssueCollection+tangled.IssueStateSuffix, rkey, tangled.IssueStateRecord{
			Type:  tangled.IssueCollection + tangled.IssueStateSuffix,
			Issue: target,
			State: state,
		})
	}
	state = tangled.PullCollection + tangled.PullStatusSuffix + "." + state
	return putRecord(ctx, atClient, did, tangled.PullCollection+tangled.PullStatusSuffix, rkey, tangled.PullStatusRecord{
		Type:   tangled.PullCollection + tangled.PullStatusSuffix,
		Pull:   target,
		Status: state,
	})
}
