package cli

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var authLoginCmd = &cobra.Command{
	Use:   "login [handle]",
	Short: "Log in to atproto via OAuth",
	Long:  `Log in to atproto via OAuth using a local browser callback.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if auth == nil {
			return fmt.Errorf("auth is not available")
		}

		identifier := ""
		if len(args) == 1 {
			identifier = args[0]
		}
		if identifier == "" {
			return fmt.Errorf("handle or DID required")
		}

		server, resultChannel, err := runCallbackServer()
		if err != nil {
			return err
		}
		defer server.Shutdown(context.Background())

		ctx := cmd.Context()
		loginURL, err := auth.StartLogin(ctx, identifier)
		if err != nil {
			return err
		}

		fmt.Println("Opening browser to complete login...")
		if err := openBrowser(loginURL); err != nil {
			fmt.Printf("Could not open browser. Open this URL manually:\n%s\n", loginURL)
		}

		select {
		case err := <-resultChannel:
			if err != nil {
				return err
			}
			fmt.Printf("Logged in as %s\n", auth.CurrentDID())
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	},
}

// runCallbackServer starts the local HTTP server that receives the OAuth
// redirect after the user approves the login in their browser.
func runCallbackServer() (*http.Server, <-chan error, error) {
	resultChannel := make(chan error, 1)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if err := auth.FinishLogin(r.Context(), r.URL.Query()); err != nil {
			resultChannel <- err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resultChannel <- nil
		fmt.Fprintln(w, "Authenticated successfully. You can close this tab.")
	})

	server := &http.Server{
		Addr:    oauthCallbackAddr,
		Handler: serveMux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			resultChannel <- fmt.Errorf("callback server: %w", err)
		}
	}()

	return server, resultChannel, nil
}

// openBrowser launches the user's default browser to url.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}
