package cli

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newBrowseCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "browse [handle/repo]",
		Short: "Open a Tangled repository in a browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTarget(cmd.Context(), args, service)
			if err != nil {
				return err
			}

			repoURL := "https://tangled.org/" + url.PathEscape(target.Handle) + "/" + url.PathEscape(target.Repo)
			if err := openURL(repoURL); err != nil {
				return fmt.Errorf("open browser: %w", err)
			}
			return nil
		},
	}
}

// openURL passes the URL as an argument instead of through a shell.
func openURL(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "--", rawURL).Start()
	case "windows":
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}
