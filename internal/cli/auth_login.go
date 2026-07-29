package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthLoginCommand(service *app.Service) *cobra.Command {
	var passwordStdin bool
	var useInsecureFileStore bool

	command := &cobra.Command{
		Use:   "login <handle> [app-password]",
		Short: "Log in to atproto via OAuth or an app password",
		Long:  `Log in with OAuth, or use an app password as the second argument for headless login.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]
			password, usePassword, err := loginPassword(args, passwordStdin, cmd.InOrStdin())
			if err != nil {
				return err
			}
			if useInsecureFileStore && !usePassword {
				return fmt.Errorf("--insecure requires an app password; pass one as the second argument or use --password-stdin")
			}
			if usePassword {
				if err := service.LoginWithPassword(cmd.Context(), identifier, password, useInsecureFileStore); err != nil {
					return err
				}
				did, err := service.CurrentDID(cmd.Context())
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "Login completed but session could not be confirmed.")
					return err
				}
				renderAuthLogin(cmd.OutOrStdout(), did)
				return nil
			}

			server, resultChannel, err := runCallbackServer(service)
			if err != nil {
				return err
			}
			defer server.Shutdown(context.Background())

			ctx := cmd.Context()
			loginURL, err := service.StartLogin(ctx, identifier)
			if err != nil {
				service.CancelLogin()
				return err
			}
			defer service.CancelLogin()

			fmt.Fprintln(cmd.OutOrStdout(), "Opening browser to complete login...")
			if err := openBrowser(loginURL); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Could not open browser. Open this URL manually:\n%s\n", loginURL)
			}

			select {
			case err := <-resultChannel:
				if err != nil {
					return err
				}
				did, err := service.CurrentDID(ctx)
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "Login completed but session could not be confirmed.")
					return err
				}
				renderAuthLogin(cmd.OutOrStdout(), did)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	command.Flags().BoolVar(&passwordStdin, "password-stdin", false, "Read the app password from standard input")
	command.Flags().BoolVar(&useInsecureFileStore, "insecure", false, "Store credentials in a file (~/.local/share/tg/credentials.json) instead of the system keyring. Requires an app password.")
	return command
}

func renderAuthLogin(writer io.Writer, did string) {
	styles := newAuthOutputStyles(isTerminal(writer))
	writeAuthHeading(writer, styles.active, "✓", "Logged in")
	writeAuthDetail(writer, styles, "DID", styles.dim.Render(did))
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

func runCallbackServer(service *app.Service) (*http.Server, <-chan error, error) {
	resultChannel := make(chan error, 1)

	listener, err := net.Listen("tcp", oauthCallbackHost+":0")
	if err != nil {
		return nil, nil, fmt.Errorf("bind callback listener: %w", err)
	}

	// Register the actual bound port as the OAuth redirect URI before starting
	// the login flow. RFC 8252 loopback redirects allow any port; binding an
	// ephemeral one avoids clashes with other local services.
	callbackURL := "http://" + listener.Addr().String() + oauthCallbackPath
	service.SetOAuthCallbackURL(callbackURL)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(oauthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.FinishLogin(r.Context(), r.URL.Query()); err != nil {
			resultChannel <- err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resultChannel <- nil
		fmt.Fprintln(w, "Authenticated successfully. You can close this tab.")
	})

	server := &http.Server{
		Handler: serveMux,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultChannel <- fmt.Errorf("callback server: %w", err)
		}
	}()

	return server, resultChannel, nil
}

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
		if isWSL() {
			cmd = "powershell.exe"
			args = []string{"-NoProfile", "-Command", fmt.Sprintf("Start-Process '%s'", url)}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}

	return exec.Command(cmd, args...).Start()
}

func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	s := strings.ToLower(string(data))
	return strings.Contains(s, "microsoft") || strings.Contains(s, "wsl")
}
