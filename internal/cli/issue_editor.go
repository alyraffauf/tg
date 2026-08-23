package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/editor"
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
	file, err := os.CreateTemp("", "tg-issue-*.md")
	if err != nil {
		return issueDraft{}, fmt.Errorf("create issue draft: %w", err)
	}
	path := file.Name()
	if _, err := file.WriteString(issueDraftTemplate); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return issueDraft{}, fmt.Errorf("write issue draft: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return issueDraft{}, fmt.Errorf("close issue draft: %w", err)
	}

	command, err := editor.CommandContext(ctx, "tg", path)
	if err != nil {
		_ = os.Remove(path)
		return issueDraft{}, fmt.Errorf("open issue editor: %w", err)
	}
	command.Stdin = input
	command.Stdout = output
	command.Stderr = errorOutput
	if err := command.Run(); err != nil {
		return issueDraft{}, fmt.Errorf("run issue editor: %w; draft saved to %s", err, path)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return issueDraft{}, fmt.Errorf("read issue draft: %w; draft saved to %s", err, path)
	}
	title, body, err := parseIssueDraft(string(contents))
	if errors.Is(err, errIssueCreationCanceled) {
		removeIssueDraft(path, errorOutput)
		return issueDraft{}, err
	}
	if err != nil {
		return issueDraft{}, fmt.Errorf("parse issue draft: %w; draft saved to %s", err, path)
	}
	return issueDraft{Title: title, Body: body, Path: path}, nil
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
	if err := os.Remove(path); err != nil {
		fmt.Fprintf(errorOutput, "warning: remove issue draft %s: %v\n", path, err)
	}
}
