package cli

import (
	"testing"

	"github.com/alyraffauf/tg/atproto"
)

func TestBuildStringItems(t *testing.T) {
	records := []atproto.RecordItem{
		{
			URI: "at://did:plc:abc/sh.tangled.string/3k2abc",
			Value: map[string]any{
				"$type":       "sh.tangled.string",
				"filename":    "test.d",
				"description": "my test string",
				"contents":    "# hello",
				"createdAt":   "2026-07-18T23:15:54+03:00",
			},
		},
		{
			URI:   "at://did:plc:abc/sh.tangled.string/3k2def",
			Value: map[string]any{"not": "a string record"},
		},
	}

	items := buildStringItems(records)

	// Records without a filename are not strings and are skipped.
	if len(items) != 1 {
		t.Fatalf("buildStringItems() returned %d items, want 1", len(items))
	}

	first := items[0]
	if first.Rkey != "3k2abc" {
		t.Errorf("Rkey = %q, want %q", first.Rkey, "3k2abc")
	}
	if first.Filename != "test.d" {
		t.Errorf("Filename = %q, want %q", first.Filename, "test.d")
	}
	if first.Description != "my test string" {
		t.Errorf("Description = %q, want %q", first.Description, "my test string")
	}
	if first.CreatedAt != "2026-07-18T23:15:54+03:00" {
		t.Errorf("CreatedAt = %q, want %q", first.CreatedAt, "2026-07-18T23:15:54+03:00")
	}
}
