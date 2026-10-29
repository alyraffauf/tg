package app

import "github.com/alyraffauf/tg/tangled"

const (
	issueCollection  = tangled.IssueCollection
	pullCollection   = tangled.PullCollection
	stringCollection = tangled.StringCollection
	sshKeyCollection = tangled.SSHKeyCollection
	repoCollection   = tangled.RepoCollection
)

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
