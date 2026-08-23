package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	issueDraftSentinel = "<!-- tg: everything below this line is ignored -->"
	issueDraftTemplate = "\n\n" + issueDraftSentinel + "\n" +
		"<!-- Enter a title above, followed by a blank line and a body. -->\n"
)

var errIssueCreationCanceled = errors.New("issue creation canceled")

type issueDraft struct {
	Title string
	Body  string
	Path  string
}

func editIssueDraft(ctx context.Context, input io.Reader, output, errorOutput io.Writer) (issueDraft, error) {
	edited, err := editDraft(ctx, "issue", "tg-issue-*.md", issueDraftTemplate, input, output, errorOutput)
	if err != nil {
		return issueDraft{}, err
	}
	title, body, err := parseIssueDraft(edited.Contents)
	if errors.Is(err, errIssueCreationCanceled) {
		removeIssueDraft(edited.Path, errorOutput)
		return issueDraft{}, err
	}
	if err != nil {
		return issueDraft{}, fmt.Errorf("parse issue draft: %w; draft saved to %s", err, edited.Path)
	}
	return issueDraft{Title: title, Body: body, Path: edited.Path}, nil
}

func parseIssueDraft(document string) (string, string, error) {
	document = strings.ReplaceAll(document, "\r\n", "\n")
	lines := strings.Split(document, "\n")
	for index, line := range lines {
		if line == issueDraftSentinel {
			lines = lines[:index]
			break
		}
	}
	document = strings.Join(lines, "\n")
	if strings.TrimSpace(document) == "" {
		return "", "", errIssueCreationCanceled
	}

	titleLine, remainder, found := strings.Cut(document, "\n")
	if !found {
		return "", "", fmt.Errorf("title must be followed by an empty line")
	}
	blankLine, body, found := strings.Cut(remainder, "\n")
	if !found {
		blankLine = remainder
		body = ""
	}
	if blankLine != "" {
		return "", "", fmt.Errorf("second line must be empty")
	}
	title := strings.TrimSpace(titleLine)
	if title == "" {
		return "", "", fmt.Errorf("title must not be empty")
	}
	body = strings.Trim(body, "\n")
	if strings.TrimSpace(body) == "" {
		return "", "", fmt.Errorf("body must not be empty")
	}
	return title, body, nil
}

func removeIssueDraft(path string, errorOutput io.Writer) {
	removeDraft(path, "issue", errorOutput)
}
