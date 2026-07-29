package cli

import (
	"bytes"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestRenderAuthStatus(t *testing.T) {
	tests := []struct {
		name   string
		result app.AuthStatusResult
		want   string
	}{
		{
			name: "not logged in",
			want: "○ Not logged in\n  Action  tg auth login <handle>\n",
		},
		{
			name: "active",
			result: app.AuthStatusResult{
				Authenticated: true,
				Status:        app.SessionStatusActive,
				Handle:        "aly.codes",
				DID:           "did:plc:abc123",
			},
			want: "✓ Authenticated\n  Account  aly.codes\n  DID  did:plc:abc123\n  Session  Active\n",
		},
		{
			name: "expired",
			result: app.AuthStatusResult{
				Authenticated: true,
				Status:        app.SessionStatusExpired,
				Handle:        "aly.codes",
			},
			want: "! Session expired\n  Account  aly.codes\n  Action  tg auth login aly.codes\n",
		},
		{
			name: "unknown",
			result: app.AuthStatusResult{
				Authenticated: true,
				Status:        app.SessionStatusUnknown,
				Handle:        "aly.codes",
			},
			want: "? Could not verify session\n  Account  aly.codes\n  Reason  Network error\n  Retry  tg auth status\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			renderAuthStatus(&output, &tt.result)
			if got := output.String(); got != tt.want {
				t.Fatalf("renderAuthStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderAuthMessages(t *testing.T) {
	tests := []struct {
		name   string
		render func(*bytes.Buffer)
		want   string
	}{
		{
			name: "login",
			render: func(output *bytes.Buffer) {
				renderAuthLogin(output, "did:plc:abc123")
			},
			want: "✓ Logged in\n  DID  did:plc:abc123\n",
		},
		{
			name: "logout",
			render: func(output *bytes.Buffer) {
				renderAuthLogout(output, &app.AuthLogoutResult{WasLoggedIn: true})
			},
			want: "✓ Logged out\n",
		},
		{
			name: "switch",
			render: func(output *bytes.Buffer) {
				renderAuthSwitch(output, &app.AuthAccountResult{
					Handle: "aly.codes",
					DID:    "did:plc:abc123",
					Method: "oauth",
				})
			},
			want: "✓ Active account changed\n  Account  aly.codes\n  DID  did:plc:abc123\n  Method  oauth\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			tt.render(&output)
			if got := output.String(); got != tt.want {
				t.Fatalf("render() = %q, want %q", got, tt.want)
			}
		})
	}
}
