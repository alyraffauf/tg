package app

import (
	"context"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/identity"
)

func TestSpindleAuthUsesManifestDIDWhenEndpointIsElsewhere(t *testing.T) {
	for _, native := range []bool{false, true} {
		service, pds, origin := migrationService(t)
		if native {
			origin.capabilities = []string{"knot-repo-token-mint"}
		}
		view := service.appview.(testAppview)
		view.repo.Value.Spindle = optionalString("did:web:ci.example")
		service.appview = view
		service.resolver = testResolver{identities: map[string]*identity.Identity{
			"did:web:ci.example": {Services: map[string]identity.ServiceEndpoint{"tangled_spindle": {Type: "TangledSpindle", URL: "https://workers.example:8443"}}},
		}}
		host, repo, err := service.pipelineTarget(context.Background(), migrationTarget())
		if err != nil || host != "https://workers.example:8443" {
			t.Fatalf("host=%s err=%v", host, err)
		}
		_, err = service.spindleToken(context.Background(), migrationTarget(), pds, host, repo, "org.tangled.event.notifyEvent")
		if err != nil {
			t.Fatal(err)
		}
		if native {
			if origin.procedureInputs[0]["aud"] != "did:web:ci.example" {
				t.Fatalf("mint=%+v", origin.procedureInputs)
			}
		} else if pds.serviceAuthAudiences[0] != "did:web:ci.example" {
			t.Fatalf("aud=%v", pds.serviceAuthAudiences)
		}
	}
}

func TestEditSpindleHostPreservesPortInDID(t *testing.T) {
	service, _, origin := migrationService(t)
	origin.records = map[string]*atproto.GetRecordOutput{tangled.RepoCollection + "/self": {CID: cidPointer(), Value: manifestValue()}}
	value := "ci.example:8443"
	if _, err := service.EditRepo(context.Background(), migrationTarget(), EditRepoInput{Spindle: &value}); err != nil {
		t.Fatal(err)
	}
	saved := origin.procedureInputs[0]["record"].(map[string]any)
	if saved["spindle"] != "did:web:ci.example%3A8443" {
		t.Fatalf("manifest=%+v", saved)
	}
}
