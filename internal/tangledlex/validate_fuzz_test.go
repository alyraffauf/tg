package tangledlex

import "testing"

func FuzzValidateRecord(f *testing.F) {
	f.Add("title", "did:plc:abc123", "2026-07-25T12:00:00Z")
	f.Fuzz(func(t *testing.T, title, repo, createdAt string) {
		if len(title)+len(repo)+len(createdAt) > 64<<10 {
			t.Skip()
		}
		_ = ValidateRecord("sh.tangled.repo.issue", RepoIssue{
			LexiconTypeID: "sh.tangled.repo.issue",
			Title:         title,
			Repo:          repo,
			CreatedAt:     createdAt,
		})
	})
}
