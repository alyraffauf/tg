package cli

import "testing"

func TestShortDate(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		want      string
	}{
		{name: "iso timestamp", timestamp: "2026-07-21T10:30:00Z", want: "2026-07-21"},
		{name: "date only", timestamp: "2026-07-21", want: "2026-07-21"},
		{name: "short string", timestamp: "2026", want: "2026"},
		{name: "empty", timestamp: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortDate(tt.timestamp); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
