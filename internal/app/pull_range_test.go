package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func publishedForkService(t *testing.T) (*Service, *testKnot, Target) {
	t.Helper()
	service, pds, origin := migrationService(t)
	pds.claims = map[string]string{"fork": "did:plc:fork"}
	view := service.appview.(testAppview)
	view.repos = map[string]*tangled.Repo{"at://did:plc:owner/org.tangled.repo.declaration/fork": {OwnerDID: "did:plc:owner", URI: "at://did:plc:fork/org.tangled.repo.manifest/self", Value: tangled.RepoMetadata{Name: optionalString("fork"), RepoDid: optionalString("did:plc:fork"), Knot: "knot.example"}}}
	view.ticket = &ticketList(true).Items[0]
	service.appview = view
	return service, origin, Target{Handle: "owner.test", Repo: "fork"}
}

// The feature is one commit beyond the published target; local main is stale.
func forkPullHistory(t *testing.T) (dir, targetTip, head string) {
	t.Helper()
	dir = t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = dir
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	git("init", "--template=", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.com")
	git("commit", "--allow-empty", "-m", "old base")
	git("switch", "-c", "target")
	git("commit", "--allow-empty", "-m", "published target")
	targetTip = git("rev-parse", "HEAD")
	git("remote", "add", "upstream", "https://target.example/repo")
	git("update-ref", "refs/remotes/upstream/main", targetTip)
	git("switch", "-c", "feature")
	git("commit", "--allow-empty", "-m", "proposal")
	head = git("rev-parse", "HEAD")
	return
}

func TestForkPullUsesPublishedTargetTipDespiteStaleLocalMain(t *testing.T) {
	dir, targetTip, head := forkPullHistory(t)
	for _, update := range []bool{false, true} {
		for _, base := range []string{"main", "upstream/main"} {
			name := "create/" + base
			if update {
				name = "update/" + base
			}
			t.Run(name, func(t *testing.T) {
				service, origin, source := publishedForkService(t)
				service.git = gitutil.NewClient(io.Discard, io.Discard)
				view := service.appview.(testAppview)
				view.refs = map[string]map[string]string{
					"did:plc:repo": {"refs/heads/main": targetTip},
					"did:plc:fork": {"refs/heads/feature": head},
				}
				view.formatHook = func(repo, from, to string) {
					if repo != "did:plc:fork" || from != targetTip || to != head {
						t.Fatalf("preflight=%s %s..%s", repo, from, to)
					}
					if len(origin.procedureMethods) != 0 {
						t.Fatal("preflight ran after a write")
					}
				}
				service.appview = view
				var err error
				if update {
					_, err = service.UpdatePullRound(context.Background(), UpdatePullRoundInput{RepoDir: dir, Target: migrationTarget(), Source: &source, Rkey: "3m2zq4c6pmk2a", Base: base, Head: "feature"})
				} else {
					_, err = service.CreatePull(context.Background(), CreatePullInput{RepoDir: dir, Target: migrationTarget(), Source: &source, Title: "PR", Base: base, Head: "feature"})
				}
				if err != nil {
					t.Fatal(err)
				}
				patch := origin.procedureInputs[len(origin.procedureInputs)-1]["patch"].(map[string]any)
				if patch["base"].(map[string]any)["oid"] != targetTip || patch["head"].(map[string]any)["oid"] != head || patch["sourceRepo"] != "did:plc:fork" {
					t.Fatalf("submitted range=%+v", patch)
				}
			})
		}
	}
}

func TestPullPreflightFailuresDoNotWrite(t *testing.T) {
	for _, update := range []bool{false, true} {
		for _, failure := range []string{"target tip missing locally", "target mirror unavailable", "patch too large"} {
			operation := "create/"
			if update {
				operation = "update/"
			}
			t.Run(operation+failure, func(t *testing.T) {
				service, origin, source := publishedForkService(t)
				git := service.git.(*testGit)
				git.pullBases["main"] = gitutil.PullBase{Revision: strings.Repeat("b", 40), Branch: "main"}
				git.commits = map[string]string{"feature": strings.Repeat("a", 40)}
				view := service.appview.(testAppview)
				published := strings.Repeat("c", 40)
				view.refs = map[string]map[string]string{
					"did:plc:repo": {"refs/heads/main": published},
					"did:plc:fork": {"refs/heads/feature": strings.Repeat("a", 40)},
				}
				cause := &atclient.APIError{StatusCode: http.StatusRequestEntityTooLarge, Name: "PatchTooLarge"}
				switch failure {
				case "target mirror unavailable":
					view.refErr = cause
				case "patch too large":
					git.commits[published] = published
					view.formatErr = cause
				}
				service.appview = view
				var err error
				if update {
					_, err = service.UpdatePullRound(context.Background(), UpdatePullRoundInput{Target: migrationTarget(), Source: &source, Rkey: "3m2zq4c6pmk2a", Base: "main", Head: "feature"})
				} else {
					_, err = service.CreatePull(context.Background(), CreatePullInput{Target: migrationTarget(), Source: &source, Title: "PR", Base: "main", Head: "feature"})
				}
				if err == nil || len(origin.procedureMethods) != 0 {
					t.Fatalf("err=%v writes=%v", err, origin.procedureMethods)
				}
				if failure == "target tip missing locally" {
					var unavailable *PullBaseUnavailableError
					if !errors.As(err, &unavailable) || unavailable.PublishedCommit != published || unavailable.LocalCommit != strings.Repeat("b", 40) {
						t.Fatalf("err=%v", err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatalf("lost error kind: %v", err)
				}
			})
		}
	}
}

func TestPullPatchErrorsIncludeStoredRange(t *testing.T) {
	service, _, _ := migrationService(t)
	view := service.appview.(testAppview)
	view.ticket = &ticketList(true).Items[0]
	failure := &atclient.APIError{StatusCode: http.StatusRequestEntityTooLarge, Name: "PatchTooLarge"}
	view.formatErr = failure
	service.appview = view
	_, err := service.PullPatch(context.Background(), migrationTarget(), "3m2zq4c6pmk2a")
	if !errors.Is(err, failure) {
		t.Fatalf("lost error kind: %v", err)
	}
	for _, detail := range []string{"did:plc:repo", strings.Repeat("b", 40), strings.Repeat("a", 40)} {
		if !strings.Contains(err.Error(), detail) {
			t.Fatalf("missing %s in %v", detail, err)
		}
	}
}
