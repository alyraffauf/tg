package app

import (
	"context"
	"testing"

	"github.com/alyraffauf/tg/tangled"
)

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

func TestResolveAuthorFallsBackWhenHandleIsInvalid(t *testing.T) {
	service := testService(nil, nil, nil)
	service.resolver = testResolver{identity: repositoryIdentity("did:plc:author", "handle.invalid", "https://knot.example")}
	author := service.resolveAuthor(context.Background(), "did:plc:author")
	if author.Handle != "did:plc:author" || author.DID != "did:plc:author" {
		t.Fatalf("author=%+v", author)
	}
}
