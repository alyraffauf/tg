package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	titledDraftSentinel = "<!-- tg: everything below this line is ignored -->"
	titledDraftTemplate = "\n\n" + titledDraftSentinel + "\n" +
		"<!-- Enter a title above, followed by a blank line and an optional body. -->\n"
	issueDraftTemplate = "\n\n" + titledDraftSentinel + "\n" +
		"<!-- Enter a title above, followed by a blank line and a body. -->\n"
)

var errTitledDraftCanceled = errors.New("title and body creation canceled")

type titledDraft struct {
	Title string
	Body  string
	Path  string
}

func editTitledDraft(ctx context.Context, kind, pattern, template string, input io.Reader, output, errorOutput io.Writer) (titledDraft, error) {
	edited, err := editDraft(ctx, kind, pattern, template, input, output, errorOutput)
	if err != nil {
		return titledDraft{}, err
	}
	title, body, err := parseTitledDraft(edited.Contents)
	if errors.Is(err, errTitledDraftCanceled) {
		removeDraft(edited.Path, kind, errorOutput)
		return titledDraft{}, err
	}
	if err != nil {
		return titledDraft{}, fmt.Errorf("parse %s draft: %w; draft saved to %s", kind, err, edited.Path)
	}
	return titledDraft{Title: title, Body: body, Path: edited.Path}, nil
}

func editIssueDraft(ctx context.Context, input io.Reader, output, errorOutput io.Writer) (titledDraft, error) {
	draft, err := editTitledDraft(ctx, "issue", "tg-issue-*.md", issueDraftTemplate, input, output, errorOutput)
	if err != nil {
		return titledDraft{}, err
	}
	if strings.TrimSpace(draft.Body) == "" {
		return titledDraft{}, fmt.Errorf("parse issue draft: body must not be empty; draft saved to %s", draft.Path)
	}
	return draft, nil
}

func parseTitledDraft(document string) (string, string, error) {
	document = strings.ReplaceAll(document, "\r\n", "\n")
	lines := strings.Split(document, "\n")
	for index, line := range lines {
		if line == titledDraftSentinel {
			lines = lines[:index]
			break
		}
	}
	document = strings.Join(lines, "\n")
	if strings.TrimSpace(document) == "" {
		return "", "", errTitledDraftCanceled
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
	return title, strings.Trim(body, "\n"), nil
}

func removeTitledDraft(path, kind string, errorOutput io.Writer) {
	removeDraft(path, kind, errorOutput)
}
