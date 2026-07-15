package cli

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var (
	repoEditDescription  string
	repoEditWebsite      string
	repoEditSpindle      string
	repoEditAddLabels    []string
	repoEditRemoveLabels []string
)

var repoEditCmd = &cobra.Command{
	Use:   "edit [handle/repo]",
	Short: "Edit a Tangled repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("description") && !cmd.Flags().Changed("website") && !cmd.Flags().Changed("spindle") && len(repoEditAddLabels) == 0 && len(repoEditRemoveLabels) == 0 {
			return fmt.Errorf("set a repository field to update")
		}
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}
		handle, name, err := resolveTarget(ctx, args)
		if err != nil {
			return err
		}
		repo, err := requireOwnedRepo(ctx, handle, name, did)
		if err != nil {
			return err
		}

		rkey := extractRKey(repo.URI)
		existing, err := atClient.GetRecord(ctx, did, "sh.tangled.repo", rkey)
		if err != nil {
			return fmt.Errorf("get repository record: %w", err)
		}
		record, err := repoRecordMap(existing.Value)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("description") {
			record["description"] = repoEditDescription
		}
		if cmd.Flags().Changed("website") {
			record["website"] = repoEditWebsite
		}
		if cmd.Flags().Changed("spindle") {
			record["spindle"] = repoEditSpindle
		}
		if len(repoEditAddLabels) > 0 || len(repoEditRemoveLabels) > 0 {
			labels := labelsFromRecord(record["labels"])
			for _, label := range repoEditAddLabels {
				labels[label] = true
			}
			for _, label := range repoEditRemoveLabels {
				delete(labels, label)
			}
			record["labels"] = labelNames(labels)
		}
		if _, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
			Repo:       did,
			Collection: "sh.tangled.repo",
			Rkey:       rkey,
			Record:     record,
		}); err != nil {
			return fmt.Errorf("edit repository: %w", err)
		}

		result := repoEditResult{URI: repo.URI, Description: repoEditDescription}
		return output(result, func(result repoEditResult) {
			fmt.Printf("Updated repository %s\n", result.URI)
		})
	},
}

type repoEditResult struct {
	URI         string `json:"uri"`
	Description string `json:"description"`
}

func init() {
	repoEditCmd.Flags().StringVarP(&repoEditDescription, "description", "d", "", "Repository description")
	repoEditCmd.Flags().StringVar(&repoEditWebsite, "website", "", "Repository website")
	repoEditCmd.Flags().StringVar(&repoEditSpindle, "spindle", "", "Repository spindle")
	repoEditCmd.Flags().StringSliceVar(&repoEditAddLabels, "add-label", nil, "Label to add")
	repoEditCmd.Flags().StringSliceVar(&repoEditRemoveLabels, "remove-label", nil, "Label to remove")
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
