package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

type createdRecordResult struct {
	Rkey string `json:"rkey"`
	URI  string `json:"uri"`
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
