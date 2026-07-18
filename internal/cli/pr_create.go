package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

const patchMimeType = "application/gzip"

var (
	prCreateTitle      string
	prCreateBody       string
	prCreateBodyFile   string
	prCreateBase       string
	prCreateHead       string
	prCreateRepo       string
	prCreateSourceRepo string
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request from the current branch",
	Long: "Create a pull request by uploading a gzipped git patch and writing a sh.tangled.repo.pull record. " +
		"By default, the current repository and branch are both the source and target repository, and origin's " +
		"default branch is the target branch. Use --repo and --source-repo for a fork-based pull request.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}

		repoDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
		head, err := prSourceBranch(ctx, repoDir)
		if err != nil {
			return err
		}
		base, err := prTargetBranch(ctx, repoDir)
		if err != nil {
			return err
		}
		body, err := commandBody(prCreateBody, prCreateBodyFile)
		if err != nil {
			return err
		}

		targetArgs := []string{}
		if prCreateRepo != "" {
			targetArgs = []string{prCreateRepo}
		}
		handle, repo, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}
		target, err := resolveRepoRecord(ctx, handle, repo)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(target.URI, "at://") {
			return fmt.Errorf("target repository %q has no strong at:// URI", repo)
		}
		source := target
		if prCreateSourceRepo != "" {
			sourceHandle, sourceName, err := parseHandleRepo(prCreateSourceRepo)
			if err != nil {
				return err
			}
			source, err = resolveRepoRecord(ctx, sourceHandle, sourceName)
			if err != nil {
				return fmt.Errorf("resolve source repository: %w", err)
			}
		}
		if source.Value.RepoDid == "" {
			return fmt.Errorf("source repository has no repo DID")
		}

		patch, err := gitutil.GeneratePatch(ctx, repoDir, base, head)
		if err != nil {
			return fmt.Errorf("generate pull request patch: %w", err)
		}
		blob, err := atClient.UploadBlob(ctx, patch, patchMimeType)
		if err != nil {
			return err
		}

		uri, err := createPullRecord(ctx, atClient, did, prCreateRecord{
			Title:         prCreateTitle,
			Body:          body,
			TargetRepoDid: target.Value.RepoDid,
			SourceRepoDid: source.Value.RepoDid,
			Base:          base,
			Head:          head,
			Patch:         blob,
		})
		if err != nil {
			return err
		}
		result := prCreateResult{URI: uri, Title: prCreateTitle, Base: base, Head: head}
		return output(result, func(created prCreateResult) {
			fmt.Printf("Created pull request %s (%s -> %s)\n", created.URI, created.Head, created.Base)
		})
	},
}

func init() {
	prCreateCmd.Flags().StringVarP(&prCreateTitle, "title", "t", "", "Pull request title")
	prCreateCmd.Flags().StringVarP(&prCreateBody, "body", "b", "", "Pull request body")
	prCreateCmd.Flags().StringVarP(&prCreateBodyFile, "body-file", "F", "", "Read pull request body from file")
	prCreateCmd.Flags().StringVarP(&prCreateBase, "base", "B", "", "Target branch (default: origin's default branch)")
	prCreateCmd.Flags().StringVarP(&prCreateHead, "head", "H", "", "Source branch (default: current branch)")
	prCreateCmd.Flags().StringVarP(&prCreateRepo, "repo", "R", "", "Target repository as handle/repo")
	prCreateCmd.Flags().StringVar(&prCreateSourceRepo, "source-repo", "", "Source repository as handle/repo (for fork-based pull requests)")
	prCreateCmd.MarkFlagRequired("title")
}

type prCreateRecord struct {
	Title         string
	Body          string
	TargetRepoDid string
	SourceRepoDid string
	Base          string
	Head          string
	Patch         *atproto.Blob
}

// pullRecord is the sh.tangled.repo.pull lexicon shape used for record writes.
type pullRecord struct {
	Type      string      `json:"$type"`
	Title     string      `json:"title"`
	Body      string      `json:"body,omitempty"`
	CreatedAt string      `json:"createdAt"`
	Target    pullTarget  `json:"target"`
	Source    pullSource  `json:"source"`
	Rounds    []pullRound `json:"rounds"`
}

type pullTarget struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

type pullSource struct {
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch"`
}

type pullRound struct {
	CreatedAt string        `json:"createdAt"`
	PatchBlob *atproto.Blob `json:"patchBlob"`
}

type prCreateResult struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
	Base  string `json:"base"`
	Head  string `json:"head"`
}

func prSourceBranch(ctx context.Context, repoDir string) (string, error) {
	if prCreateHead != "" {
		return prCreateHead, nil
	}
	branch, err := gitutil.CurrentBranch(ctx, repoDir)
	if err != nil {
		return "", fmt.Errorf("determine source branch: %w", err)
	}
	return branch, nil
}

func prTargetBranch(ctx context.Context, repoDir string) (string, error) {
	if prCreateBase != "" {
		return prCreateBase, nil
	}
	branch, err := gitutil.DefaultBranch(ctx, repoDir)
	if err != nil {
		return "", fmt.Errorf("determine target branch; set --base explicitly: %w", err)
	}
	return branch, nil
}

func createPullRecord(ctx context.Context, atClient *atproto.ATProto, did string, input prCreateRecord) (string, error) {
	record := newPullRecord(input, time.Now().UTC())
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: "sh.tangled.repo.pull",
		Rkey:       string(syntax.NewTIDNow(0)),
		Record:     record,
	})
	if err != nil {
		return "", fmt.Errorf("create pull request record: %w", err)
	}
	return uri, nil
}

func newPullRecord(input prCreateRecord, createdAt time.Time) pullRecord {
	now := createdAt.Format(time.RFC3339)
	return pullRecord{
		Type:      "sh.tangled.repo.pull",
		Title:     input.Title,
		Body:      input.Body,
		CreatedAt: now,
		Target: pullTarget{
			Repo:   input.TargetRepoDid,
			Branch: input.Base,
		},
		Source: pullSource{
			Repo:   input.SourceRepoDid,
			Branch: input.Head,
		},
		Rounds: []pullRound{{
			CreatedAt: now,
			PatchBlob: input.Patch,
		}},
	}
}
