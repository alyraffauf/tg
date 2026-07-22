package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

const bytesPerMiB = 1 << 20

// maxStringContents caps string contents at 100 MiB, a client-side sanity
// limit for a text record.
const maxStringContents = 100 * bytesPerMiB

func newStringCreateCommand(service *app.Service) *cobra.Command {
	var description, filenameFlag string

	command := &cobra.Command{
		Use:   "create [<file>]",
		Short: "Create a string on your Tangled account",
		Long: `Create a string on your Tangled account.

Contents are read from the given file, or from standard input if no file
is given (or the file is "-"). When reading from standard input,
--filename is required. Contents must be valid UTF-8, at most 100 MiB.
Requires authentication (run "tg auth login" first).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			contents, filename, err := stringContents(cmd.InOrStdin(), args, filenameFlag)
			if err != nil {
				return err
			}

			result, err := service.CreateString(ctx, app.CreateStringInput{
				Filename:    filename,
				Description: description,
				Contents:    contents,
			})
			if err != nil {
				return err
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
			return "", "", fmt.Errorf("--filename is required when reading from standard input")
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
