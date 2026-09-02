package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

// EditRepoInput configures repository edits. Pointer fields are nil when the
// corresponding flag was not set.
type EditRepoInput struct {
	Description  *string
	Website      *string
	Spindle      *string
	AddLabels    []string
	RemoveLabels []string
}

// EditRepo patches repository fields on the authenticated user's repo t.
func (s *Service) EditRepo(ctx context.Context, t Target, in EditRepoInput) (*RepoEditResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	rkey := extractRKey(repo.URI)
	if err := updateRecord(ctx, atClient, did, repoCollection, rkey, func(value any) (map[string]any, error) {
		record, err := repoRecordMap(value)
		if err != nil {
			return nil, err
		}
		if in.Description != nil {
			setOrDelete(record, "description", *in.Description)
		}
		if in.Website != nil {
			setOrDelete(record, "website", *in.Website)
		}
		if in.Spindle != nil {
			setOrDelete(record, "spindle", *in.Spindle)
		}
		if len(in.AddLabels) > 0 || len(in.RemoveLabels) > 0 {
			labels := labelsFromRecord(record["labels"])
			for _, label := range in.AddLabels {
				labels[label] = true
			}
			for _, label := range in.RemoveLabels {
				delete(labels, label)
			}
			record["labels"] = labelNames(labels)
		}
		return record, nil
	}); err != nil {
		return nil, fmt.Errorf("edit repository: %w", err)
	}
	result := &RepoEditResult{URI: repo.URI}
	if in.Description != nil {
		result.Description = *in.Description
	}
	return result, nil
}

func setOrDelete(record map[string]any, field, value string) {
	if value == "" {
		delete(record, field)
		return
	}
	record[field] = value
}

func repoRecordMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode repository record: %w", err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decode repository record: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("repository record is not an object")
	}
	return record, nil
}

func labelsFromRecord(value any) map[string]bool {
	labels := make(map[string]bool)
	values, ok := value.([]any)
	if !ok {
		return labels
	}
	for _, value := range values {
		if label, ok := value.(string); ok {
			labels[label] = true
		}
	}
	return labels
}

func labelNames(labels map[string]bool) []string {
	names := make([]string, 0, len(labels))
	for label := range labels {
		names = append(names, label)
	}
	sort.Strings(names)
	return names
}
