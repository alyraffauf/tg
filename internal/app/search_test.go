package app

import (
	"context"
	"testing"

	"github.com/alyraffauf/tg/tangled"
)

func TestSearch(t *testing.T) {
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.appview = testAppview{stars: 73, search: &tangled.SearchResult{Hits: []tangled.SearchHit{{
		URI:   "at://did:plc:repo/org.tangled.repo.manifest/self",
		NSID:  "org.tangled.repo.manifest",
		Score: 3.5,
		Value: map[string]any{"$type": "org.tangled.repo.manifest", "declaration": map[string]any{"owner": "did:plc:owner", "name": "example"}, "description": "Example"},
	}}}}

	got, err := service.SearchRepos(context.Background(), "example", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].URI != "at://did:plc:repo/org.tangled.repo.manifest/self" || got.Items[0].Stars != 73 {
		t.Fatalf("Search() = %+v", got)
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	if _, err := service.SearchRepos(context.Background(), "", 10); err == nil {
		t.Fatal("SearchRepos() accepted an empty query")
	}
}
