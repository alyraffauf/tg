package gitutil

import "testing"

func FuzzParseRepoCandidate(f *testing.F) {
	f.Add("https://tangled.org/owner.test/repo")
	f.Add("git@knot.example:did:plc:abc123")
	f.Fuzz(func(t *testing.T, remote string) {
		if len(remote) > 16<<10 {
			t.Skip()
		}
		_, _ = parseRepoCandidate(remote)
	})
}
