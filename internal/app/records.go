package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
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
func putRecord(ctx context.Context, atClient pdsClient, did, collection, rkey string, record any) error {
	if _, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo: did, Collection: collection, Rkey: rkey, Record: record,
	}); err != nil {
		return err
	}
	return nil
}

// editRecord fetches an existing record, applies the provided title and/or
// body patches (nil leaves the field untouched), and writes it back.
func editRecord(ctx context.Context, atClient pdsClient, did, collection, rkey string, title, body *string) error {
	found, err := atClient.GetRecord(ctx, did, collection, rkey)
	if err != nil {
		return fmt.Errorf("get existing record: %w", err)
	}

	record, err := editLexiconRecord(collection, found.Value, title, body)
	if err != nil {
		return err
	}
	return putRecord(ctx, atClient, did, collection, rkey, record)
}

func editLexiconRecord(collection string, value any, title, body *string) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode existing record: %w", err)
	}

	switch collection {
	case issueCollection:
		var record tangledlex.RepoIssue
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("decode existing issue: %w", err)
		}
		if title != nil {
			record.Title = *title
		}
		if body != nil {
			record.Body = body
		}
		return record, nil
	case pullCollection:
		var record tangledlex.RepoPull
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("decode existing pull request: %w", err)
		}
		if title != nil {
			record.Title = *title
		}
		if body != nil {
			record.Body = body
		}
		return record, nil
	default:
		return nil, fmt.Errorf("editing records in collection %q is not supported", collection)
	}
}

// putState writes an issue.state or pull.status record keyed by rkey. state
// is the bare verb ("open"/"closed"/"merged"); the collection-specific suffix
// is applied here.
func putState(ctx context.Context, atClient pdsClient, did, rkey, collection, target, state string) error {
	if collection == tangled.IssueCollection {
		state = tangled.IssueCollection + tangled.IssueStateSuffix + "." + state
		return putRecord(ctx, atClient, did, tangled.IssueCollection+tangled.IssueStateSuffix, rkey, tangledlex.RepoIssueState{
			LexiconTypeID: tangled.IssueCollection + tangled.IssueStateSuffix,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			Issue:         target,
			State:         state,
		})
	}
	state = tangled.PullCollection + tangled.PullStatusSuffix + "." + state
	return putRecord(ctx, atClient, did, tangled.PullCollection+tangled.PullStatusSuffix, rkey, tangledlex.RepoPullStatus{
		LexiconTypeID: tangled.PullCollection + tangled.PullStatusSuffix,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Pull:          target,
		Status:        state,
	})
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
