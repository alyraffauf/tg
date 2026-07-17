package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var authLoginPasswordStdin bool

var authLoginCmd = &cobra.Command{
	Use:   "login <handle> [app-password]",
	Short: "Log in to atproto via OAuth or an app password",
	Long:  `Log in with OAuth, or use an app password as the second argument for headless login.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if auth == nil {
			return fmt.Errorf("auth is not available")
		}

		identifier := args[0]
		password, usePassword, err := loginPassword(args, authLoginPasswordStdin, cmd.InOrStdin())
		if err != nil {
			return err
		}
		if usePassword {
			if err := auth.LoginWithPassword(cmd.Context(), identifier, password); err != nil {
				return err
			}
			fmt.Printf("Logged in as %s\n", auth.CurrentDID())
			return nil
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

func init() {
	authLoginCmd.Flags().BoolVar(&authLoginPasswordStdin, "password-stdin", false, "Read the app password from standard input")
}

func loginPassword(args []string, fromStdin bool, stdin io.Reader) (string, bool, error) {
	if !fromStdin {
		if len(args) < 2 {
			return "", false, nil
		}
		return args[1], true, nil
	}
	if len(args) == 2 {
		return "", false, fmt.Errorf("app password argument and --password-stdin cannot be used together")
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", false, fmt.Errorf("read app password from stdin: %w", err)
	}
	password := strings.TrimSpace(string(data))
	if password == "" {
		return "", false, fmt.Errorf("app password from stdin is empty")
	}
	return password, true, nil
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
