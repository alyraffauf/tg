package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/app"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

func TestAuthTokenOutputModes(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	const token = "test-access-token"
	store := atproto.NewKeyringStore()
	err := store.SavePasswordSession(context.Background(), atclient.PasswordSessionData{
		AccountDID: syntax.DID("did:plc:aaaabbbbccccddddeeeeffff"), Host: "https://pds.example", AccessToken: token,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, jsonMode := range []bool{false, true} {
		var stdout bytes.Buffer
		service := app.NewWithStreams("https://appview.example", &stdout, &bytes.Buffer{})
		root := NewRoot(service)
		root.SetOut(&stdout)
		args := []string{"auth", "token"}
		if jsonMode {
			args = append(args, "--json")
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if jsonMode {
			var result app.AccessTokenResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Token != token {
				t.Fatalf("result=%+v", result)
			}
		} else if stdout.String() != token+"\n" {
			t.Fatalf("stdout=%q", stdout.String())
		}
	}
}
