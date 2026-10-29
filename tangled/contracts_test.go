package tangled

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestTicketAuthorFilterAppliesBeforeLimitAcrossPages(t *testing.T) {
	pages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		if r.URL.Path != "/xrpc/org.tangled.repo.listIssues" || r.URL.Query().Get("author") != "" {
			t.Errorf("request=%s", r.URL)
		}
		author := "did:plc:other"
		cursor := ",\"cursor\":\"next\""
		key := "first"
		if pages == 2 {
			author = "did:plc:alice"
			cursor = ""
			key = "second"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"items":[{"uri":"at://did:plc:repo/org.tangled.track.ticket/%s","cid":"cid","author":"%s","state":"open","value":{"$type":"org.tangled.track.ticket","title":{"$type":"org.tangled.markup.markdown#inline","text":"Title"},"state":"open","createdAt":"2026-10-01T00:00:00Z"}}]%s}`, key, author, cursor)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	result, err := client.ListIssues(context.Background(), "did:plc:repo", ListOpts{Author: "did:plc:alice", MaxItems: 1})
	if err != nil || pages != 2 || len(result.Items) != 1 || result.Items[0].Author != "did:plc:alice" {
		t.Fatalf("result=%+v pages=%d err=%v", result, pages, err)
	}
}
func TestPullFilterMapsMergedToCompletedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "completed" || r.URL.Query().Get("state") != "" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[{"uri":"at://did:plc:repo/org.tangled.track.ticket/key","author":"did:plc:alice","state":"completed","value":{"$type":"org.tangled.track.ticket","targetBranch":"main","title":{"text":"Title"},"state":"completed"}}]}`)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	result, err := client.ListPulls(context.Background(), "did:plc:repo", ListOpts{State: "merged"})
	if err != nil || len(result.Items) != 1 || result.Items[0].State != "merged" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
func TestDetailedTicketUsesInitialAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/org.tangled.track.getTicketView" || r.URL.Query().Get("uri") == "" {
			t.Errorf("request=%s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"uri":"at://did:plc:repo/org.tangled.track.ticket/key","cid":"cid","author":{"did":"did:plc:alice"},"title":{"text":"Title"},"state":"closed","createdAt":"2026-10-01T00:00:00Z"}`)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	item, err := client.GetTicket(context.Background(), "at://did:plc:repo/org.tangled.track.ticket/key")
	if err != nil || item.Author != "did:plc:alice" || item.URI != "at://did:plc:repo/org.tangled.track.ticket/key" || item.CID != "cid" {
		t.Fatalf("item=%+v err=%v", item, err)
	}
}
func TestFormatPatchStreamsTextAndMergeUsesServedLegacyName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Path != "/xrpc/org.tangled.temp.git.formatPatch" || r.URL.Query().Get("from") != "base" || r.URL.Query().Get("to") != "head" {
				t.Errorf("request=%s", r.URL)
			}
			fmt.Fprint(w, "From commit Mon Sep 17 00:00:00 2001\n")
			return
		}
		if r.URL.Path != "/xrpc/sh.tangled.git.mergeCommit" || r.Header.Get("Authorization") != "Bearer push-token" {
			t.Errorf("merge=%s auth=%s", r.URL, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	patch, err := client.FormatPatch(context.Background(), "did:plc:repo", "base", "head")
	if err != nil || len(patch) == 0 {
		t.Fatalf("patch=%q err=%v", patch, err)
	}
	if err := client.Merge(context.Background(), "push-token", map[string]string{"style": "rebase"}); err != nil {
		t.Fatal(err)
	}
}
func FuzzManifestResponse(f *testing.F) {
	f.Add([]byte(`{"uri":"at://did:plc:repo/org.tangled.repo.manifest/self","value":{"$type":"org.tangled.repo.manifest","declaration":{"owner":"did:plc:alice","name":"Repo"}}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		var repo Repo
		_ = json.Unmarshal(data, &repo)
	})
}

func TestFormatPatchPreservesMirrorLagError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"RevisionNotFound","message":"head is not mirrored yet"}`)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	_, err := client.FormatPatch(context.Background(), "did:plc:fork", "base", "head")
	var apiError *atclient.APIError
	if !errors.As(err, &apiError) || apiError.Name != "RevisionNotFound" || apiError.StatusCode != 404 {
		t.Fatalf("err=%v", err)
	}
}
