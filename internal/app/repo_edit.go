package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/alyraffauf/tg/spindle"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// EditRepoInput configures repository edits. Pointer fields are nil when the
// corresponding flag was not set.
type EditRepoInput struct {
	ContributionPolicy *string
	Description        *string
	Website            *string
	Spindle            *string
	AddLabels          []string
	RemoveLabels       []string
}

// EditRepo patches repository fields on the authenticated user's repo t.
func (s *Service) EditRepo(ctx context.Context, t Target, in EditRepoInput) (*RepoEditResult, error) {
	if in.ContributionPolicy != nil {
		if err := validateContributionPolicy(*in.ContributionPolicy); err != nil {
			return nil, err
		}
	}
	_, actor, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, actor)
	if err != nil {
		return nil, err
	}
	warnings, err := s.editManifest(ctx, repo, func(record map[string]any) error {
		if in.Description != nil {
			setOrDelete(record, "description", *in.Description)
		}
		if in.Website != nil {
			setOrDelete(record, "website", *in.Website)
		}
		if in.Spindle != nil {
			value := *in.Spindle
			if value != "" && !strings.HasPrefix(value, "did:") {
				var err error
				value, err = spindle.ServiceDID(value)
				if err != nil {
					return err
				}
			}
			if value != "" {
				if _, err := syntax.ParseDID(value); err != nil {
					return err
				}
			}
			setOrDelete(record, "spindle", value)
		}
		if in.ContributionPolicy != nil {
			record["contributionPolicy"] = *in.ContributionPolicy
		}
		if len(in.AddLabels) > 0 || len(in.RemoveLabels) > 0 {
			topics := labelsFromRecord(record["topics"])
			for _, topic := range in.AddLabels {
				topics[topic] = true
			}
			for _, topic := range in.RemoveLabels {
				delete(topics, topic)
			}
			record["topics"] = labelNames(topics)
		}
		delete(record, "defaultBranch")
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := &RepoEditResult{URI: repo.URI, Warnings: warnings}
	if in.Description != nil {
		result.Description = *in.Description
	}
	return result, nil
}
func validateContributionPolicy(policy string) error {
	if policy != "" && policy != "anyone" && policy != "members" && policy != "collaborators" {
		return fmt.Errorf("contribution policy must be anyone, members or collaborators")
	}
	return nil
}
func (s *Service) editManifest(ctx context.Context, repo *tangled.Repo, mutate func(map[string]any) error) ([]string, error) {
	found, err := s.knot.NewPublic(repo.Value.Knot).GetRecord(ctx, stringValue(repo.Value.RepoDid), tangled.RepoCollection, "self")
	if err != nil {
		return nil, err
	}
	if found.CID == nil {
		return nil, fmt.Errorf("manifest response omitted CID")
	}
	record, err := repoRecordMap(found.Value)
	if err != nil {
		return nil, err
	}
	if err := mutate(record); err != nil {
		return nil, err
	}
	_, warnings, err := s.socialProcedure(ctx, repo, "com.atproto.repo.putRecord", map[string]any{"collection": tangled.RepoCollection, "rkey": "self", "swapRecord": found.CID.String(), "record": record})
	return warnings, err
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
