package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

const bytesPerMiB = 1 << 20

// maxStringContents caps string contents at 100 MiB, a client-side sanity
// limit for a text record.
const maxStringContents = 100 * bytesPerMiB

var (
	stringCreateDescription string
	stringCreateFilename    string
)

var stringCreateCmd = &cobra.Command{
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

		contents, filename, err := stringContents(os.Stdin, args, stringCreateFilename)
		if err != nil {
			return err
		}

		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}

		rkey := string(syntax.NewTIDNow(0))
		uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
			Repo:       did,
			Collection: stringCollection,
			Rkey:       rkey,
			Record: stringRecord{
				Type:        stringCollection,
				Filename:    filename,
				Description: stringCreateDescription,
				Contents:    contents,
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			},
		})
		if err != nil {
			return fmt.Errorf("create string: %w", err)
		}

		return output(createdRecordResult{Rkey: rkey, URI: uri}, func(result createdRecordResult) {
			fmt.Printf("Created string %s\n", result.URI)
		})
	},
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

func init() {
	stringCreateCmd.Flags().StringVarP(&stringCreateDescription, "description", "d", "", "Description of the string")
	stringCreateCmd.Flags().StringVarP(&stringCreateFilename, "filename", "f", "", "Filename for the string (defaults to the file's basename)")
}
