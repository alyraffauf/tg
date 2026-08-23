package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/editor"
)

const (
	editorBytesPerMiB  = 1 << 20
	maxEditorDraftSize = 100 * editorBytesPerMiB
)

type editedDraft struct {
	Contents string
	Path     string
}

func editDraft(ctx context.Context, kind, pattern, template string, input io.Reader, output, errorOutput io.Writer) (editedDraft, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return editedDraft{}, fmt.Errorf("create %s draft: %w", kind, err)
	}
	path := file.Name()
	if _, err := file.WriteString(template); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return editedDraft{}, fmt.Errorf("write %s draft: %w", kind, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return editedDraft{}, fmt.Errorf("close %s draft: %w", kind, err)
	}

	command, err := editor.CommandContext(ctx, "tg", path)
	if err != nil {
		_ = os.Remove(path)
		return editedDraft{}, fmt.Errorf("open %s editor: %w", kind, err)
	}
	command.Stdin = input
	command.Stdout = output
	command.Stderr = errorOutput
	if err := command.Run(); err != nil {
		return editedDraft{}, fmt.Errorf("run %s editor: %w; draft saved to %s", kind, err, path)
	}

	contents, err := readDraftContents(path, kind, maxEditorDraftSize)
	if err != nil {
		return editedDraft{}, err
	}
	return editedDraft{Contents: contents, Path: path}, nil
}

func readDraftContents(path, kind string, sizeMax int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read %s draft: %w; draft saved to %s", kind, err, path)
	}
	defer file.Close()

	contents, err := io.ReadAll(io.LimitReader(file, sizeMax+1))
	if err != nil {
		return "", fmt.Errorf("read %s draft: %w; draft saved to %s", kind, err, path)
	}
	if int64(len(contents)) > sizeMax {
		return "", fmt.Errorf("read %s draft: contents exceed the %d MiB limit; draft saved to %s", kind, sizeMax/editorBytesPerMiB, path)
	}
	return string(contents), nil
}

func removeDraft(path, kind string, errorOutput io.Writer) {
	if err := os.Remove(path); err != nil {
		fmt.Fprintf(errorOutput, "warning: remove %s draft %s: %v\n", kind, path, err)
	}
}
