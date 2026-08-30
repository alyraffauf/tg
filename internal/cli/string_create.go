package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/alyraffauf/tg/internal/app"
	xterm "github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

const bytesPerMiB = 1 << 20

// maxStringContents caps string contents at 100 MiB, a client-side sanity
// limit for a text record.
const maxStringContents = 100 * bytesPerMiB

var isTerminalInput = func(input io.Reader) bool {
	file, ok := input.(*os.File)
	return ok && xterm.IsTerminal(file.Fd())
}

type stringCreateService interface {
	CreateString(context.Context, app.CreateStringInput) (*app.CreatedRecordResult, error)
}

func newStringCreateCommand(service stringCreateService) *cobra.Command {
	var description, filenameFlag string

	command := &cobra.Command{
		Use:   "create [<file>]",
		Short: "Create a string on your Tangled account",
		Long: `Create a string on your Tangled account.

Contents are read from the given file. With no file, tg opens $EDITOR when
standard input is a terminal and otherwise reads standard input. The file
"-" always reads standard input. --filename is required when no file is
given. Contents must be valid UTF-8, at most 100 MiB.
Requires authentication (run "tg auth login" first).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			input := cmd.InOrStdin()
			draft := editedDraft{}
			var contents, filename string
			var err error
			if len(args) == 0 && isTerminalInput(input) {
				if filenameFlag == "" {
					return fmt.Errorf("provide --filename when composing in an editor")
				}
				draft, err = editDraft(ctx, "string", "tg-string-*", "", input, cmd.OutOrStdout(), cmd.ErrOrStderr())
				if err != nil {
					return err
				}
				if draft.Contents == "" {
					removeDraft(draft.Path, "string", cmd.ErrOrStderr())
					fmt.Fprintln(cmd.ErrOrStderr(), "String creation canceled.")
					return nil
				}
				contents, filename, err = validateStringContents(draft.Contents, filenameFlag)
			} else {
				contents, filename, err = stringContents(input, args, filenameFlag)
			}
			if err != nil {
				if draft.Path != "" {
					return fmt.Errorf("%w; draft saved to %s", err, draft.Path)
				}
				return err
			}

			result, err := service.CreateString(ctx, app.CreateStringInput{
				Filename:    filename,
				Description: description,
				Contents:    contents,
			})
			if err != nil {
				if draft.Path != "" {
					return fmt.Errorf("%w; draft saved to %s", err, draft.Path)
				}
				return err
			}
			if draft.Path != "" {
				removeDraft(draft.Path, "string", cmd.ErrOrStderr())
			}
			return output(cmd, result, func(result *app.CreatedRecordResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created string %s\n", result.URI)
			})
		},
	}
	command.Flags().StringVarP(&description, "description", "d", "", "Description of the string")
	command.Flags().StringVarP(&filenameFlag, "filename", "f", "", "Filename for the string (defaults to the file's basename)")
	return command
}

// stringContents reads string contents from the file named in args (or stdin
// when absent or "-") and resolves the filename: the flag wins, then the
// file's basename. Contents must be non-empty, valid UTF-8, and within
// maxStringContents.
func stringContents(stdin io.Reader, args []string, filenameFlag string) (contents, filename string, err error) {
	if len(args) == 0 || args[0] == "-" {
		if filenameFlag == "" {
			return "", "", fmt.Errorf("provide --filename when reading from standard input")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", "", fmt.Errorf("read standard input: %w", err)
		}
		contents, filename = string(data), filenameFlag
	} else {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return "", "", fmt.Errorf("read file %q: %w", args[0], err)
		}
		contents, filename = string(data), filenameFlag
		if filename == "" {
			filename = filepath.Base(args[0])
		}
	}

	return validateStringContents(contents, filename)
}

func validateStringContents(contents, filename string) (string, string, error) {
	if contents == "" {
		return "", "", fmt.Errorf("contents must not be empty")
	}
	if len(contents) > maxStringContents {
		return "", "", fmt.Errorf("contents exceed the %d MiB limit", maxStringContents/bytesPerMiB)
	}
	if !utf8.ValidString(contents) {
		return "", "", fmt.Errorf("contents must be valid UTF-8")
	}
	return contents, filename, nil
}
