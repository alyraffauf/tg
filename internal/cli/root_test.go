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
	create, _, err := NewRoot(&app.Service{}).Find([]string{"repo", "create"})
	if err != nil {
		t.Fatalf("find repo create command: %v", err)
	}
	flag := create.Flags().Lookup("ssh-port")
	if flag == nil || flag.Usage != "SSH port for cloning from or pushing to the selected Knot" {
		t.Fatalf("ssh-port flag = %+v", flag)
	}
	if flag.DefValue != "22" {
		t.Fatalf("ssh-port default = %q, want 22", flag.DefValue)
	}
	if err := create.Flags().Set("ssh-port", "2200"); err != nil {
		t.Fatalf("set ssh-port: %v", err)
	}
	if got := flag.Value.String(); got != "2200" {
		t.Fatalf("ssh-port = %q, want 2200", got)
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
