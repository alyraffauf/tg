package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestNewRootCreatesIndependentCommandState(t *testing.T) {
	firstRoot := NewRoot(&app.Service{})
	secondRoot := NewRoot(&app.Service{})

	firstCreate, _, err := firstRoot.Find([]string{"repo", "create"})
	if err != nil {
		t.Fatalf("find first repo create command: %v", err)
	}
	secondCreate, _, err := secondRoot.Find([]string{"repo", "create"})
	if err != nil {
		t.Fatalf("find second repo create command: %v", err)
	}
	if firstCreate == secondCreate {
		t.Fatal("NewRoot reused the repo create command")
	}

	if err := firstCreate.Flags().Set("description", "first root"); err != nil {
		t.Fatalf("set first root flag: %v", err)
	}
	if got := secondCreate.Flags().Lookup("description").Value.String(); got != "" {
		t.Fatalf("second root inherited description %q", got)
	}
}

func TestListCommandsSupportFilters(t *testing.T) {
	root := NewRoot(&app.Service{})

	for _, path := range [][]string{{"issue", "list"}, {"pr", "list"}} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %s: %v", strings.Join(path, " "), err)
		}
		for _, name := range []string{"author", "state", "limit", "order"} {
			if command.Flags().Lookup(name) == nil {
				t.Fatalf("%s is missing --%s", strings.Join(path, " "), name)
			}
		}
	}

	command := newIssueListCommand(&app.Service{})
	if err := command.Flags().Set("limit", "0"); err != nil {
		t.Fatalf("set limit: %v", err)
	}
	flags := listFlags{limit: 0}
	if _, err := flags.options(command); err == nil {
		t.Fatal("issue list accepted an explicit zero limit")
	}
}

func TestVersionFlag(t *testing.T) {
	var output bytes.Buffer
	err := ExecuteWith([]string{"--version"}, nil, &output, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("ExecuteWith(--version) error = %v", err)
	}
	if want := "tg version " + version + "\n"; output.String() != want {
		t.Errorf("--version output = %q, want %q", output.String(), want)
	}
}

func TestExecuteWithRendersErrors(t *testing.T) {
	var errorOutput bytes.Buffer
	err := ExecuteWith([]string{"issue", "edit", "abc123"}, nil, &bytes.Buffer{}, &errorOutput)
	if err == nil {
		t.Fatal("ExecuteWith() returned nil error")
	}

	const want = "Error: provide --title or --body\n"
	if got := errorOutput.String(); got != want {
		t.Errorf("error output = %q, want %q", got, want)
	}
}

func TestExecuteWithRendersPreCommandErrors(t *testing.T) {
	var errorOutput bytes.Buffer
	err := ExecuteWith([]string{"--appview"}, nil, &bytes.Buffer{}, &errorOutput)
	if err == nil {
		t.Fatal("ExecuteWith() returned nil error")
	}

	if got := errorOutput.String(); !strings.Contains(got, "Error: flag --appview requires a value\n") {
		t.Errorf("error output = %q", got)
	}
}

func TestRepoCreateSSHPortHelp(t *testing.T) {
	create, _, err := newRoot(&app.Service{}, "configured.example", "2200", "ssh").Find([]string{"repo", "create"})
	if err != nil {
		t.Fatalf("find repo create command: %v", err)
	}
	flag := create.Flags().Lookup("ssh-port")
	if flag == nil || flag.Usage != "SSH port for cloning from or pushing to an explicitly selected Knot (overrides config file and TG_SSH_PORT)" {
		t.Fatalf("ssh-port flag = %+v", flag)
	}
	if flag.DefValue != "2200" {
		t.Fatalf("ssh-port default = %q, want 2200", flag.DefValue)
	}
	if err := create.Flags().Set("ssh-port", "2222"); err != nil {
		t.Fatalf("set ssh-port: %v", err)
	}
	if got := flag.Value.String(); got != "2222" {
		t.Fatalf("ssh-port = %q, want 2222", got)
	}
}

func TestRepoCreateRejectsMalformedSSHPortForExplicitKnot(t *testing.T) {
	command := newRepoCreateCommand(&app.Service{}, "configured.example", "not-a-port", "ssh")
	if err := command.Flags().Set("clone", "true"); err != nil {
		t.Fatalf("set clone flag: %v", err)
	}
	err := command.RunE(command, []string{"example"})
	if err == nil || !strings.Contains(err.Error(), `invalid SSH port "not-a-port"`) {
		t.Fatalf("repo create error = %v", err)
	}
}

func TestParseRepoCreateSSHPortIgnoresProxyRemote(t *testing.T) {
	port, err := parseRepoCreateSSHPort("not-a-port", "", "ssh", true, "")
	if err != nil {
		t.Fatalf("parseRepoCreateSSHPort() error = %v", err)
	}
	if port != 0 {
		t.Fatalf("port = %d, want 0", port)
	}
}

func TestRepoCreateKnotFlag(t *testing.T) {
	create, _, err := newRoot(&app.Service{}, "configured.example", "22", "ssh").Find([]string{"repo", "create"})
	if err != nil {
		t.Fatalf("find repo create command: %v", err)
	}
	flag := create.Flags().Lookup("knot")
	if flag == nil {
		t.Fatal("repo create has no knot flag")
	}
	if flag.DefValue != "configured.example" || flag.Usage != "Knot host to provision and optionally push to (overrides TG_KNOT, config, and automatic discovery)" {
		t.Fatalf("knot flag = %+v", flag)
	}
	if err := create.Flags().Set("knot", "flag.example"); err != nil {
		t.Fatalf("set knot flag: %v", err)
	}
	if got := flag.Value.String(); got != "flag.example" {
		t.Fatalf("explicit knot = %q, want flag.example", got)
	}
	if NewRoot(&app.Service{}).PersistentFlags().Lookup("knot") != nil {
		t.Fatal("knot flag must not be global")
	}
}

func TestNewRootCreatesIndependentStateCommands(t *testing.T) {
	firstRoot := NewRoot(&app.Service{})
	secondRoot := NewRoot(&app.Service{})

	firstClose, _, err := firstRoot.Find([]string{"issue", "close"})
	if err != nil {
		t.Fatalf("find first issue close command: %v", err)
	}
	secondClose, _, err := secondRoot.Find([]string{"issue", "close"})
	if err != nil {
		t.Fatalf("find second issue close command: %v", err)
	}
	if firstClose == secondClose {
		t.Fatal("NewRoot reused the issue close command")
	}

	if err := firstClose.Flags().Set("repo", "first/repository"); err != nil {
		t.Fatalf("set first issue close repo: %v", err)
	}
	if got := secondClose.Flags().Lookup("repo").Value.String(); got != "" {
		t.Fatalf("second root inherited repo %q", got)
	}
}

func TestParseFlagSettings(t *testing.T) {
	flags, err := parseFlagSettings([]string{
		"--appview", "https://flag.example",
		"--account=flag.example",
		"--config", "/tmp/tg.toml",
		"--", "--account", "ignored",
	})
	if err != nil {
		t.Fatalf("parseFlagSettings() error = %v", err)
	}
	if flags.Appview != "https://flag.example" || !flags.AppviewSet {
		t.Fatalf("unexpected appview settings: %+v", flags)
	}
	if flags.Account != "flag.example" || !flags.AccountSet {
		t.Fatalf("unexpected account settings: %+v", flags)
	}
	if flags.ConfigPath != "/tmp/tg.toml" || !flags.ConfigSet {
		t.Fatalf("unexpected config settings: %+v", flags)
	}
}

func TestParseFlagSettingsRejectsMissingValue(t *testing.T) {
	if _, err := parseFlagSettings([]string{"--appview"}); err == nil {
		t.Fatal("parseFlagSettings() accepted a missing value")
	}
}
