package cli

import (
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestPRUpdateAcceptsBaseFlag(t *testing.T) {
	command := newPRUpdateCommand(&app.Service{})
	flag := command.Flags().Lookup("base")
	if flag == nil {
		t.Fatal("pr update has no --base flag")
	}
	if flag.Shorthand != "B" {
		t.Fatalf("--base shorthand = %q, want B", flag.Shorthand)
	}
}
