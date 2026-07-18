package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// manCmd is hidden from `tg --help`: it is a build-time tool the Nix
// derivation invokes to generate man pages, not a user-facing command (cf.
// `gh`, which produces man pages via its Makefile rather than a visible
// subcommand).
var manCmd = &cobra.Command{
	Use:    "man [directory]",
	Short:  "Generate man pages",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE: func(_ *cobra.Command, args []string) error {
		dir := args[0]
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create man page directory: %w", err)
		}
		header := &doc.GenManHeader{
			Title:   "tg",
			Section: "1",
			Source:  "tg",
		}
		return doc.GenManTree(rootCmd, header, dir)
	},
}
