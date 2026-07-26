package app

import (
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
)

func TestCanonicalRepoItemsPrefersNamedRecord(t *testing.T) {
	items := []tangled.Repo{
		{
			URI:   "at://did:plc:owner/sh.tangled.repo/3kold",
			Value: tangledlex.Repo{Name: optionalString("current"), RepoDid: optionalString("did:plc:repo")},
		},
		{
			URI:   "at://did:plc:owner/sh.tangled.repo/current",
			Value: tangledlex.Repo{Name: optionalString("current"), RepoDid: optionalString("did:plc:repo")},
		},
		{
			URI:   "at://did:plc:owner/sh.tangled.repo/other",
			Value: tangledlex.Repo{Name: optionalString("other"), RepoDid: optionalString("did:plc:other")},
		},
	}

	got := canonicalRepoItems(items)
	if len(got) != 2 {
		t.Fatalf("canonicalRepoItems() returned %d items, want 2", len(got))
	}
	if got[0].URI != "at://did:plc:owner/sh.tangled.repo/current" {
		t.Fatalf("canonicalRepoItems() first URI = %q, want canonical record", got[0].URI)
	}
}

func TestCanonicalRepoForAliasReturnsNamedRecord(t *testing.T) {
	alias := tangled.Repo{
		URI:   "at://did:plc:owner/sh.tangled.repo/old-name",
		Value: tangledlex.Repo{Name: optionalString("current-name"), RepoDid: optionalString("did:plc:repo")},
	}
	current := tangled.Repo{
		URI:   "at://did:plc:owner/sh.tangled.repo/current-name",
		Value: tangledlex.Repo{Name: optionalString("current-name"), RepoDid: optionalString("did:plc:repo")},
	}

	got := canonicalRepoForAlias([]tangled.Repo{alias, current}, alias)
	if got.URI != current.URI {
		t.Fatalf("canonicalRepoForAlias() URI = %q, want %q", got.URI, current.URI)
	}
}

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
