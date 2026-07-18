package cli

import "testing"

func TestForkSourceURL(t *testing.T) {
	tests := []struct {
		name    string
		knot    string
		repoDID string
		want    string
	}{
		{"bare host", "knot.gaze.systems", "did:plc:abc", "https://knot.gaze.systems/did:plc:abc"},
		{"https host", "https://knot.gaze.systems", "did:plc:abc", "https://knot.gaze.systems/did:plc:abc"},
		{"trailing slash", "https://knot.gaze.systems/", "did:plc:abc", "https://knot.gaze.systems/did:plc:abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := forkSourceURL(tt.knot, tt.repoDID); got != tt.want {
				t.Fatalf("forkSourceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
