package tangled

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestRepositorySearchFiltersBeforeLimitAndFollowsCursor(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/xrpc/org.tangled.search.query" || r.URL.Query().Get("nsid") != RepoCollection || r.URL.Query().Get("limit") != "2" {
			t.Errorf("request=%s", r.URL)
		}
		if calls == 1 {
			fmt.Fprint(w, `{"hits":[{"uri":"first"}],"cursor":"next"}`)
		} else {
			if r.URL.Query().Get("cursor") != "next" {
				t.Errorf("cursor=%s", r.URL)
			}
			fmt.Fprint(w, `{"hits":[{"uri":"second"},{"uri":"third"}]}`)
		}
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	result, err := client.Search(context.Background(), "forge", 2)
	if err != nil || calls != 2 || len(result.Hits) != 2 || result.Hits[1].URI != "second" {
		t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
	}
}
