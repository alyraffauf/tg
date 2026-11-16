package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLoginPassword(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		stdinFlag bool
		stdin     string
		want      string
		wantUse   bool
		wantErr   bool
	}{
		{"oauth", []string{"alice.example"}, false, "", "", false, false},
		{"argument", []string{"alice.example", "app-pass"}, false, "", "app-pass", true, false},
		{"stdin", []string{"alice.example"}, true, "app-pass\n", "app-pass", true, false},
		{"both", []string{"alice.example", "app-pass"}, true, "other", "", false, true},
		{"empty stdin", []string{"alice.example"}, true, "\n", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, use, err := loginPassword(tt.args, tt.stdinFlag, strings.NewReader(tt.stdin))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want || use != tt.wantUse {
				t.Fatalf("got (%q, %v), want (%q, %v)", got, use, tt.want, tt.wantUse)
			}
		})
	}
}

func TestParseOAuthCallback(t *testing.T) {
	const callback = "http://127.0.0.1:12345/oauth/callback"
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"authorized", callback + "?state=login-state&code=auth-code\n", false},
		{"whitespace", " \t" + callback + "?state=login-state&code=auth-code\r\n", false},
		{"userinfo", "http://u@127.0.0.1:12345/oauth/callback?state=s&code=c", true},
		{"fragment", callback + "?state=s&code=c#fragment", true},
		{"without newline", callback + "?state=login-state&code=auth-code", false},
		{"authorization denied", callback + "?state=login-state&error=access_denied\n", false},
		{"empty input", "", true},
		{"wrong port", "http://127.0.0.1:54321/oauth/callback?state=s&code=c", true},
		{"wrong host", "http://example.com:12345/oauth/callback?state=s&code=c", true},
		{"missing state", callback + "?code=c", true},
		{"missing result", callback + "?state=s", true},
		{"invalid query", callback + "?state=s&code=%zz", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parseOAuthCallback(tt.input, callback)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && query.Get("state") != "login-state" {
				t.Fatalf("callback state = %q", query.Get("state"))
			}
		})
	}
}

func TestPromptOAuthCallbackRetriesInvalidPaste(t *testing.T) {
	const callback = "http://127.0.0.1:12345/oauth/callback"
	var output bytes.Buffer
	query, err := promptOAuthCallback(strings.NewReader("wrong URL\n"+callback+"?state=s&code=c\n"), &output, callback)
	if err != nil {
		t.Fatal(err)
	}
	if query.Get("code") != "c" || query.Get("state") != "s" {
		t.Fatalf("unexpected callback: %v", query)
	}
	if strings.Count(output.String(), "Callback URL: ") != 2 {
		t.Fatalf("expected another prompt after invalid input: %s", output.String())
	}
}

func TestPromptOAuthCallbackEOF(t *testing.T) {
	for _, input := range []string{"", "wrong URL", "\n"} {
		_, err := promptOAuthCallback(strings.NewReader(input), io.Discard, "http://127.0.0.1:12345/oauth/callback")
		if !errors.Is(err, io.EOF) {
			t.Fatalf("input %q: got %v, want EOF", input, err)
		}
	}
}
