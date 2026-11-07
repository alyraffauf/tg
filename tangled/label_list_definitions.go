package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// labelDefinitionPageSize keeps definition listings to one request for
// typical repositories while staying under the lexicon's maximum of 1000.
const labelDefinitionPageSize = 100

// LabelDefinition is one org.tangled.label.definition record owned by a
// repository. Names are not unique and Color is optional.
type LabelDefinition struct {
	URI   string
	CID   string
	Key   string
	Name  string
	Color string
}

// LabelDefinitionList is every label definition of a repository, in creation
// order, plus records the appview returned but tg could not decode.
type LabelDefinitionList struct {
	Items    []LabelDefinition
	Warnings []RecordWarning
}

// ListLabelDefinitions follows every page of org.tangled.label.listDefinitions
// for repoDID.
func (t *Tangled) ListLabelDefinitions(ctx context.Context, repoDID string) (*LabelDefinitionList, error) {
	var warnings []RecordWarning
	items, err := fetchAllPages(ctx, 0, func(ctx context.Context, cursor string) ([]LabelDefinition, *string, error) {
		params := map[string]any{"subject": repoDID, "limit": labelDefinitionPageSize, "order": "asc"}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var page struct {
			Items []struct {
				URI   string          `json:"uri"`
				CID   string          `json:"cid"`
				Value json.RawMessage `json:"value"`
			} `json:"items"`
			Cursor *string `json:"cursor"`
		}
		if err := t.Client.Get(ctx, syntax.NSID("org.tangled.label.listDefinitions"), params, &page); err != nil {
			return nil, nil, err
		}
		definitions := make([]LabelDefinition, 0, len(page.Items))
		for _, item := range page.Items {
			definition, err := decodeLabelDefinition(repoDID, item.URI, item.Value)
			if err != nil {
				warnings = append(warnings, RecordWarning{URI: item.URI, Error: fmt.Sprintf("invalid label definition: %v", err)})
				continue
			}
			definition.CID = item.CID
			definitions = append(definitions, definition)
		}
		return definitions, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list label definitions: %w", err)
	}
	return &LabelDefinitionList{Items: items, Warnings: warnings}, nil
}

func decodeLabelDefinition(repoDID, uri string, raw json.RawMessage) (LabelDefinition, error) {
	parsed, err := syntax.ParseATURI(uri)
	if err != nil {
		return LabelDefinition{}, err
	}
	if parsed.Authority().String() != repoDID || parsed.Collection().String() != LabelDefinitionCollection || parsed.RecordKey() == "" {
		return LabelDefinition{}, fmt.Errorf("URI is not a definition in repository %s", repoDID)
	}
	var value struct {
		Type  string `json:"$type"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return LabelDefinition{}, err
	}
	if value.Type != "" && value.Type != LabelDefinitionCollection {
		return LabelDefinition{}, fmt.Errorf("unexpected record type %q", value.Type)
	}
	if value.Name == "" {
		return LabelDefinition{}, fmt.Errorf("definition has no name")
	}
	return LabelDefinition{URI: uri, Key: parsed.RecordKey().String(), Name: value.Name, Color: value.Color}, nil
}
