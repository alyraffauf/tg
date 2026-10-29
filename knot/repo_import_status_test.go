package knot

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestImportStatusContract(t *testing.T) {
	for _, status := range []string{"fetching", "present", "partial", "failed", "", "unrecognized"} {
		t.Run(status, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/xrpc/org.tangled.repo.getImportStatus" || r.URL.Query().Get("repo") != "did:plc:repo" {
					t.Errorf("unexpected request %s", r.URL)
				}
				fmt.Fprintf(w, `{"status":%q,"reason":"missing LFS objects"}`, status)
			}))
			defer server.Close()
			client := &Client{APIClient: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
			out, err := client.GetImportStatus(context.Background(), "did:plc:repo")
			if status == "unrecognized" {
				if err == nil {
					t.Fatal("unknown status accepted")
				}
				return
			}
			if err != nil || out.Status != status || calls != 1 {
				t.Fatalf("%+v %v calls=%d", out, err, calls)
			}
		})
	}
}
