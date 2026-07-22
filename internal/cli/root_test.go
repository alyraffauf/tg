package cli

import (
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
