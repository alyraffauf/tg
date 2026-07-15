package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

const patchMimeType = "application/vnd.git.patch+gzip"

var (
	prCreateTitle    string
	prCreateBody     string
	prCreateBodyFile string
	prCreateBase     string
	prCreateHead     string
	prCreateRepo     string
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request from the current branch",
	Long: "Create a pull request by uploading a gzipped git patch and writing a sh.tangled.repo.pull record. " +
		"The source and target repository are the same. By default, the current branch is the source and " +
		"origin's default branch is the target. Use --repo to target a different Tangled repository.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if auth == nil || !auth.IsAuthenticated() {
			return fmt.Errorf("not logged in; run \"tg auth login\" first")
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
		body, err := prBody()
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
		target, err := findTargetRepo(ctx, handle, repo)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(target.URI, "at://") {
			return fmt.Errorf("target repository %q has no strong at:// URI", repo)
		}

		patch, err := gitutil.GeneratePatch(ctx, repoDir, base, head)
		if err != nil {
			return fmt.Errorf("generate pull request patch: %w", err)
		}
		pds, err := auth.APIClient(ctx)
		if err != nil {
			return fmt.Errorf("get auth client: %w", err)
		}
		atClient := &atproto.ATProto{Client: pds}
		blob, err := atClient.UploadBlob(ctx, patch, patchMimeType)
		if err != nil {
			return err
		}

		uri, err := createPullRecord(ctx, atClient, auth.CurrentDID().String(), prCreateRecord{
			Title:      prCreateTitle,
			Body:       body,
			TargetRepo: target.URI,
			Base:       base,
			Head:       head,
			Patch:      blob,
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
	prCreateCmd.MarkFlagRequired("title")
}

type prCreateRecord struct {
	Title      string
	Body       string
	TargetRepo string
	Base       string
	Head       string
	Patch      *atproto.Blob
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

func prBody() (string, error) {
	if prCreateBodyFile == "" {
		return prCreateBody, nil
	}
	if prCreateBody != "" {
		return "", fmt.Errorf("--body and --body-file cannot be used together")
	}
	body, err := os.ReadFile(prCreateBodyFile)
	if err != nil {
		return "", fmt.Errorf("read pull request body: %w", err)
	}
	return string(body), nil
}

func findTargetRepo(ctx context.Context, handle, name string) (*tangled.Repo, error) {
	ident, err := resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", handle, err)
	}

	uri := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, name)
	if repo, err := client.GetRepo(ctx, uri); err == nil {
		if repo.URI == "" {
			repo.URI = uri
		}
		return repo, nil
	} else if !isNotFoundError(err) {
		return nil, fmt.Errorf("get repository %q: %w", name, err)
	}

	repos, err := client.ListRepos(ctx, ident.DID.String())
	if err != nil {
		return nil, fmt.Errorf("list repositories for %q: %w", handle, err)
	}
	for _, repo := range repos.Items {
		if repo.Value.Name == name || strings.HasSuffix(repo.URI, "/"+name) {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("repo %q not found for handle %q", name, handle)
}

func createPullRecord(ctx context.Context, atClient *atproto.ATProto, did string, input prCreateRecord) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	record := pullRecord{
		Type:      "sh.tangled.repo.pull",
		Title:     input.Title,
		Body:      input.Body,
		CreatedAt: now,
		Target:    pullTarget{Repo: input.TargetRepo, Branch: input.Base},
		Source:    pullSource{Branch: input.Head},
		Rounds: []pullRound{{
			CreatedAt: now,
			PatchBlob: input.Patch,
		}},
	}
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
