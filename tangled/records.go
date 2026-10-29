package tangled

import (
	"bytes"
	"encoding/json"
)

const (
	FeedCommentCollection = "org.tangled.feed.comment"
	IssueCollection       = "org.tangled.track.ticket"
	PullCollection        = IssueCollection
	StringCollection      = "org.tangled.snippet.string"
	SSHKeyCollection      = "org.tangled.key.sshKey"
	RepoCollection        = "org.tangled.repo.manifest"
	DeclarationCollection = "org.tangled.repo.declaration"
	AttestationCollection = "org.tangled.graph.attestation"
)

type Markdown struct {
	Type string `json:"$type"`
	Text string `json:"text"`
}

type Ticket struct {
	Type         string    `json:"$type"`
	Title        Markdown  `json:"title"`
	Body         *Markdown `json:"body,omitempty"`
	State        string    `json:"state"`
	TargetBranch string    `json:"targetBranch,omitempty"`
	CreatedAt    string    `json:"createdAt"`
}

// UnmarshalJSON accepts historical knot tickets whose title predates inline markup.
// Marshaling always uses the current structured title.
func (t *Ticket) UnmarshalJSON(data []byte) error {
	type ticketFields Ticket
	var fields ticketFields
	var wire struct {
		*ticketFields
		Title json.RawMessage `json:"title"`
	}
	wire.ticketFields = &fields
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	wire.Title = bytes.TrimSpace(wire.Title)
	var text string
	if len(wire.Title) > 0 && wire.Title[0] == '"' {
		if err := json.Unmarshal(wire.Title, &text); err != nil {
			return err
		}
		fields.Title = Markdown{Type: "org.tangled.markup.markdown#inline", Text: text}
	} else if len(wire.Title) > 0 {
		if err := json.Unmarshal(wire.Title, &fields.Title); err != nil {
			return err
		}
	}
	*t = Ticket(fields)
	return nil
}

type Manifest struct {
	Type        string `json:"$type"`
	Declaration struct {
		Owner string `json:"owner"`
		Name  string `json:"name"`
	} `json:"declaration"`
	Description        *string  `json:"description,omitempty"`
	Website            *string  `json:"website,omitempty"`
	Spindle            *string  `json:"spindle,omitempty"`
	Topics             []string `json:"topics,omitempty"`
	ContributionPolicy string   `json:"contributionPolicy,omitempty"`
	CreatedAt          string   `json:"createdAt"`
}

type Patch struct {
	URI   string `json:"uri"`
	CID   string `json:"cid"`
	Value struct {
		Ticket string `json:"ticket"`
		Base   struct {
			OID string `json:"oid"`
		} `json:"base"`
		Head struct {
			OID string `json:"oid"`
		} `json:"head"`
		Source *struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
		} `json:"source,omitempty"`
		CreatedAt string `json:"createdAt"`
	} `json:"value"`
}
