package tangled

const (
	IssueCollection  = "sh.tangled.repo.issue"
	PullCollection   = "sh.tangled.repo.pull"
	StringCollection = "sh.tangled.string"
	SSHKeyCollection = "sh.tangled.publicKey"
	RepoCollection   = "sh.tangled.repo"
	IssueStateSuffix = ".state"
	PullStatusSuffix = ".status"
)

// IssueCommentRecord is the value of a sh.tangled.repo.issue.comment record.
type IssueCommentRecord struct {
	Type      string `json:"$type"`
	Issue     string `json:"issue"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// PullCommentRecord is the value of a sh.tangled.repo.pull.comment record.
type PullCommentRecord struct {
	Type      string `json:"$type"`
	Pull      string `json:"pull"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// IssueStateRecord is the value of a sh.tangled.repo.issue.state record.
type IssueStateRecord struct {
	Type  string `json:"$type"`
	Issue string `json:"issue"`
	State string `json:"state"`
}

// PullStatusRecord is the value of a sh.tangled.repo.pull.status record.
type PullStatusRecord struct {
	Type   string `json:"$type"`
	Pull   string `json:"pull"`
	Status string `json:"status"`
}

// SSHKeyRecord is the value of a sh.tangled.publicKey record.
type SSHKeyRecord struct {
	Type      string `json:"$type"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// StringRecord is the value of a sh.tangled.string record.
type StringRecord struct {
	Type        string `json:"$type"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Contents    string `json:"contents"`
	CreatedAt   string `json:"createdAt"`
}
