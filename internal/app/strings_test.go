package app

import (
	"testing"

	"github.com/alyraffauf/tg/tangled"

	"github.com/alyraffauf/tg/atproto"
)

func TestDecodeStringRecord(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    tangled.StringRecord
		wantErr bool
	}{
		{
			name: "valid record",
			value: map[string]any{
				"$type":       "sh.tangled.string",
				"filename":    "hello.md",
				"description": "a greeting",
				"contents":    "# hello",
				"createdAt":   "2026-07-18T23:15:54+03:00",
			},
			want: tangled.StringRecord{
				Type:        "sh.tangled.string",
				Filename:    "hello.md",
				Description: "a greeting",
				Contents:    "# hello",
				CreatedAt:   "2026-07-18T23:15:54+03:00",
			},
		},
		{
			name: "record without description",
			value: map[string]any{
				"$type":     "sh.tangled.string",
				"filename":  "bare.md",
				"contents":  "no description",
				"createdAt": "2026-07-18T12:00:00Z",
			},
			want: tangled.StringRecord{
				Type:      "sh.tangled.string",
				Filename:  "bare.md",
				Contents:  "no description",
				CreatedAt: "2026-07-18T12:00:00Z",
			},
		},
		{
			name: "empty filename",
			value: map[string]any{
				"$type":     "sh.tangled.string",
				"filename":  "",
				"contents":  "empty filename",
				"createdAt": "2026-07-18T12:00:00Z",
			},
			want: tangled.StringRecord{
				Type:      "sh.tangled.string",
				Filename:  "",
				Contents:  "empty filename",
				CreatedAt: "2026-07-18T12:00:00Z",
			},
		},
		{
			name:    "non-object value",
			value:   "not an object",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := decodeStringRecord(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeStringRecord() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if record.Filename != tt.want.Filename {
				t.Errorf("Filename = %q, want %q", record.Filename, tt.want.Filename)
			}
			if record.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", record.Description, tt.want.Description)
			}
			if record.Contents != tt.want.Contents {
				t.Errorf("Contents = %q, want %q", record.Contents, tt.want.Contents)
			}
			if record.CreatedAt != tt.want.CreatedAt {
				t.Errorf("CreatedAt = %q, want %q", record.CreatedAt, tt.want.CreatedAt)
			}
		})
	}
}

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
