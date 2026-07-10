package cli

import (
	"encoding/json"
	"os"
)

// output dispatches structured data to JSON (when --json is set) or to
// a human-readable renderer.
func output[T any](data T, human func(T)) error {
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	human(data)
	return nil
}

type author struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
}

type issueItem struct {
	Rkey         string `json:"rkey"`
	URI          string `json:"uri"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Author       author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	CommentCount int64  `json:"commentCount"`
}

type pullItem struct {
	Rkey         string `json:"rkey"`
	URI          string `json:"uri"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Author       author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	CommentCount int64  `json:"commentCount"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch"`
}

type repoItem struct {
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Author      string `json:"author"`
	Knot        string `json:"knot"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
	RepoDid     string `json:"repoDid,omitempty"`
}

type sshKeyItem struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
}

type issueViewResult struct {
	Rkey      string `json:"rkey"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Author    author `json:"author"`
	CreatedAt string `json:"createdAt"`
}

type prViewResult struct {
	Rkey         string `json:"rkey"`
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	Author       author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch"`
}

type repoCreateResult struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	URI    string `json:"uri"`
	Knot   string `json:"knot"`
	Cloned bool   `json:"cloned"`
	Pushed bool   `json:"pushed"`
}

type repoCloneResult struct {
	Handle      string `json:"handle"`
	Repo        string `json:"repo"`
	Destination string `json:"destination"`
}

type sshKeyAddResult struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type prCheckoutResult struct {
	Rkey      string `json:"rkey"`
	Branch    string `json:"branch"`
	Directory string `json:"directory"`
}

type authStatusResult struct {
	Authenticated bool   `json:"authenticated"`
	DID           string `json:"did,omitempty"`
	Handle        string `json:"handle,omitempty"`
}
