package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthLoginCommand(service *app.Service) *cobra.Command {
	var passwordStdin bool
	var useInsecureFileStore bool

	command := &cobra.Command{
		Use:   "login <handle> [app-password]",
		Short: "Log in with OAuth or an app password",
		Long:  `Log in with OAuth, or pass an app password. If the browser cannot be opened, paste the callback URL after authorizing. Use --insecure to store credentials in plaintext when no OS keyring is available.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]
			password, usePassword, err := loginPassword(args, passwordStdin, cmd.InOrStdin())
			if err != nil {
				return err
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

			var finishOnce sync.Once
			var finishErr error
			finishLogin := func(ctx context.Context, query url.Values) error {
				finishOnce.Do(func() { finishErr = service.FinishLogin(ctx, query) })
				return finishErr
			}
			server, resultChannel, err := runCallbackServer(service, finishLogin)
			if err != nil {
				return err
			}
			defer server.Shutdown(context.Background())

			ctx := cmd.Context()
			loginURL, err := service.StartLogin(ctx, identifier, useInsecureFileStore)
			if err != nil {
				service.CancelLogin()
				return err
			}
			defer service.CancelLogin()

			fmt.Fprintln(cmd.OutOrStdout(), "Opening browser to complete login...")
			browserResult := make(chan error, 1)
			go func(result chan<- error) { result <- openBrowser(loginURL) }(browserResult)
			manualCallback := make(chan url.Values, 1)

		waitForLogin:
			for {
				select {
				case err := <-browserResult:
					browserResult = nil
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Could not open browser. Open this URL manually:\n%s\n", loginURL)
						fmt.Fprintln(cmd.ErrOrStderr(), "After authorizing, copy the full callback URL from your browser's address bar and paste it here.")
						go func() {
							query, err := promptOAuthCallback(cmd.InOrStdin(), cmd.ErrOrStderr(), "http://"+server.Addr+oauthCallbackPath)
							if errors.Is(err, io.EOF) {
								return
							}
							if err != nil {
								select {
								case resultChannel <- err:
								case <-ctx.Done():
								}
								return
							}
							manualCallback <- query
						}()
					}
				case query := <-manualCallback:
					if err := finishLogin(ctx, query); err != nil {
						return err
					}
					break waitForLogin
				case err := <-resultChannel:
					if err != nil {
						return err
					}
					break waitForLogin
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			did, err := service.CurrentDID(ctx)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Login completed but session could not be confirmed.")
				return err
			}
			renderAuthLogin(cmd.OutOrStdout(), did)
			return nil
		},
	}
	command.Flags().BoolVar(&passwordStdin, "password-stdin", false, "Read the app password from standard input")
	command.Flags().BoolVar(&useInsecureFileStore, "insecure", false, "Store credentials in ~/.local/share/tg/credentials.json instead of the system keyring.")
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

func promptOAuthCallback(reader io.Reader, writer io.Writer, expectedURL string) (url.Values, error) {
	buffered := bufio.NewReader(reader)
	for {
		fmt.Fprint(writer, "Callback URL: ")
		line, readErr := buffered.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("read callback URL: %w", readErr)
		}
		if readErr == io.EOF && strings.TrimSpace(line) == "" {
			return nil, io.EOF
		}
		query, err := parseOAuthCallback(line, expectedURL)
		if err == nil {
			return query, nil
		}
		fmt.Fprintln(writer, err)
		if readErr == io.EOF {
			return nil, io.EOF
		}
	}
}

func parseOAuthCallback(line, expectedURL string) (url.Values, error) {
	callback, err := url.Parse(strings.TrimSpace(line))
	if err != nil {
		return nil, fmt.Errorf("parse callback URL: %w", err)
	}
	expected, err := url.Parse(expectedURL)
	if err != nil {
		return nil, err
	}
	if callback.Scheme != expected.Scheme || callback.Host != expected.Host || callback.Path != expected.Path || callback.User != nil || callback.Fragment != "" {
		return nil, fmt.Errorf("callback URL must start with %s", expectedURL)
	}
	query, err := url.ParseQuery(callback.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("parse callback query: %w", err)
	}
	if query.Get("state") == "" || (query.Get("code") == "" && query.Get("error") == "") {
		return nil, fmt.Errorf("callback URL must include state and code or error parameters")
	}
	return query, nil
}

func runCallbackServer(service *app.Service, finishLogin func(context.Context, url.Values) error) (*http.Server, chan error, error) {
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
		if err := finishLogin(r.Context(), r.URL.Query()); err != nil {
			select {
			case resultChannel <- err:
			default:
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		select {
		case resultChannel <- nil:
		default:
		}
		fmt.Fprintln(w, "Authenticated successfully. You can close this tab.")
	})

	server := &http.Server{
		Addr:    listener.Addr().String(),
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
			args = []string{"-NoProfile", "-Command", "Start-Process -FilePath $args[0]", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}

	return exec.Command(cmd, args...).Run()
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
