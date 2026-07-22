package app

// Author is the owner or creator of a record, resolved from a DID.
type Author struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
}

// Item is a listing entry for an issue or a pull request. SourceBranch and
// TargetBranch are only populated (and only emitted as JSON) for pulls.
type Item struct {
	Rkey         string `json:"rkey"`
	URI          string `json:"uri"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Author       Author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	CommentCount int64  `json:"commentCount"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
}

// RepoItem is a single repository in a listing or view.
type RepoItem struct {
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Author      string `json:"author"`
	Knot        string `json:"knot"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
	RepoDid     string `json:"repoDid,omitempty"`
}

// SSHKeyItem is one SSH public key in a listing.
type SSHKeyItem struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
}

// StringItem is one tangled string in a listing.
type StringItem struct {
	Rkey        string `json:"rkey"`
	URI         string `json:"uri"`
	Filename    string `json:"filename"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// StringViewResult is the full view of a single string.
type StringViewResult struct {
	Rkey        string `json:"rkey"`
	URI         string `json:"uri"`
	Filename    string `json:"filename"`
	Author      Author `json:"author"`
	Description string `json:"description,omitempty"`
	Contents    string `json:"contents"`
	CreatedAt   string `json:"createdAt"`
}

// ViewResult is a single issue or pull request. SourceBranch and
// TargetBranch are only populated (and only emitted as JSON) for pulls.
type ViewResult struct {
	Rkey         string `json:"rkey"`
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	Author       Author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
}

// CreatedRecordResult is returned by any operation that creates a record.
type CreatedRecordResult struct {
	Rkey string `json:"rkey"`
	URI  string `json:"uri"`
}

// DeletedRecordResult is returned by any operation that deletes a record.
type DeletedRecordResult struct {
	Rkey string `json:"rkey"`
}

// StateResult is returned by issue/PR state changes (close, reopen, merge).
type StateResult struct {
	Rkey  string `json:"rkey"`
	State string `json:"state"`
}

// RepoCreateResult is returned by repository creation.
type RepoCreateResult struct {
	Handle        string   `json:"handle"`
	Name          string   `json:"name"`
	URI           string   `json:"uri"`
	Knot          string   `json:"knot"`
	Cloned        bool     `json:"cloned"`
	Pushed        bool     `json:"pushed"`
	DefaultBranch string   `json:"defaultBranch,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

// RepoCloneResult is returned by repository cloning.
type RepoCloneResult struct {
	Handle      string `json:"handle"`
	Repo        string `json:"repo"`
	Destination string `json:"destination"`
}

// RepoEditResult is returned by repository edits.
type RepoEditResult struct {
	URI         string `json:"uri"`
	Description string `json:"description"`
}

// RepoDeleteResult is returned by repository deletion.
type RepoDeleteResult struct {
	URI string `json:"uri"`
}

// RepoDefaultBranchResult is returned by setting a repo's default branch.
type RepoDefaultBranchResult struct {
	URI    string `json:"uri"`
	Branch string `json:"branch"`
}

// RepoForkResult is returned by repository forking.
type RepoForkResult struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	URI    string `json:"uri"`
	Knot   string `json:"knot"`
}

// PRCreateResult is returned by pull request creation.
type PRCreateResult struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
	Base  string `json:"base"`
	Head  string `json:"head"`
}

// PRCheckoutResult is returned by pull request checkout.
type PRCheckoutResult struct {
	Rkey   string `json:"rkey"`
	Branch string `json:"branch"`
}

// SSHKeyAddResult is returned by SSH key addition.
type SSHKeyAddResult struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// AuthStatusResult is returned by an auth status probe.
type AuthStatusResult struct {
	Authenticated bool   `json:"authenticated"`
	Status        string `json:"status,omitempty"`
	DID           string `json:"did,omitempty"`
	Handle        string `json:"handle,omitempty"`
}

// AuthLogoutResult is returned by logout. WasLoggedIn reports whether a
// session existed and was cleared; it is false when there was nothing to
// log out (not a failure — the command still exits 0).
type AuthLogoutResult struct {
	WasLoggedIn bool `json:"wasLoggedIn"`
}

// AuthAccountResult is one account in an account listing or switch.
type AuthAccountResult struct {
	Active bool   `json:"active"`
	DID    string `json:"did"`
	Handle string `json:"handle"`
	Method string `json:"method"`
}
