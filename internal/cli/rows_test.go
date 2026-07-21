package cli

import (
	"testing"

	"github.com/alyraffauf/tg/tangled"
)

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

func TestExtractDID(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{name: "record uri", uri: "at://did:plc:abc123/sh.tangled.repo.issue/3kdui", want: "did:plc:abc123"},
		{name: "bare did", uri: "did:plc:abc123", want: "did:plc:abc123"},
		{name: "trailing slash", uri: "at://did:plc:abc123/", want: "did:plc:abc123"},
		{name: "empty", uri: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractDID(tt.uri); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRKey(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{name: "record uri", uri: "at://did:plc:abc123/sh.tangled.repo.issue/3kdui", want: "3kdui"},
		{name: "bare rkey", uri: "3kdui", want: "3kdui"},
		{name: "trailing slash", uri: "at://did:plc:abc123/", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRKey(tt.uri); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeIssue(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    recordView
		wantErr bool
	}{
		{
			name: "full record",
			raw:  `{"title":"Bug report","body":"details","createdAt":"2026-07-18T12:00:00Z"}`,
			want: recordView{Title: "Bug report", Body: "details", CreatedAt: "2026-07-18T12:00:00Z"},
		},
		{
			name: "title only",
			raw:  `{"title":"Bug report"}`,
			want: recordView{Title: "Bug report"},
		},
		{name: "invalid json", raw: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeIssue([]byte(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDecodePull(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    recordView
		wantErr bool
	}{
		{
			name: "full record",
			raw: `{"title":"Add feature","body":"details","createdAt":"2026-07-18T12:00:00Z",` +
				`"source":{"branch":"feature"},"target":{"branch":"main"}}`,
			want: recordView{
				Title:        "Add feature",
				Body:         "details",
				CreatedAt:    "2026-07-18T12:00:00Z",
				SourceBranch: "feature",
				TargetBranch: "main",
			},
		},
		{
			name: "title only",
			raw:  `{"title":"Add feature"}`,
			want: recordView{Title: "Add feature"},
		},
		{name: "invalid json", raw: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodePull([]byte(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFindByRKey(t *testing.T) {
	items := []tangled.ListItem{
		{URI: "at://did:plc:abc123/sh.tangled.repo.issue/3kdui"},
		{URI: "at://did:plc:abc123/sh.tangled.repo.issue/9xyz"},
	}

	tests := []struct {
		name    string
		rkey    string
		wantURI string
		wantErr bool
	}{
		{name: "found", rkey: "3kdui", wantURI: items[0].URI},
		{name: "found second", rkey: "9xyz", wantURI: items[1].URI},
		{name: "not found", rkey: "missing", wantErr: true},
		{name: "no partial match", rkey: "xyz", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findByRKey(items, tt.rkey, "issue")
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got.URI != tt.wantURI {
				t.Fatalf("got %q, want %q", got.URI, tt.wantURI)
			}
		})
	}
}
