package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	commentDraftSentinel = "<!-- tg: everything below this line is ignored -->"
	commentDraftTemplate = "\n" + commentDraftSentinel + "\n" +
		"<!-- Enter a non-empty comment above. -->\n"
)

var errCommentCreationCanceled = errors.New("comment creation canceled")

type commentDraft struct {
	Body string
	Path string
}

func editCommentDraft(ctx context.Context, input io.Reader, output, errorOutput io.Writer) (commentDraft, error) {
	edited, err := editDraft(ctx, "comment", "tg-comment-*.md", commentDraftTemplate, input, output, errorOutput)
	if err != nil {
		return commentDraft{}, err
	}
	body := parseCommentDraft(edited.Contents)
	if strings.TrimSpace(body) == "" {
		removeCommentDraft(edited.Path, errorOutput)
		return commentDraft{}, errCommentCreationCanceled
	}
	return commentDraft{Body: body, Path: edited.Path}, nil
}

func parseCommentDraft(document string) string {
	document = strings.ReplaceAll(document, "\r\n", "\n")
	lines := strings.Split(document, "\n")
	for index, line := range lines {
		if line == commentDraftSentinel {
			lines = lines[:index]
			break
		}
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

func removeCommentDraft(path string, errorOutput io.Writer) {
	removeDraft(path, "comment", errorOutput)
}

func commentSubmissionError(err error, draft commentDraft) error {
	if draft.Path == "" {
		return err
	}
	return fmt.Errorf("%w; draft saved to %s", err, draft.Path)
}
