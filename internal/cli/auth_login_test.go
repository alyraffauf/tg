package cli

import (
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
