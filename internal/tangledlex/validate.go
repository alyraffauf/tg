package tangledlex

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
	"unicode/utf8"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ValidateRecord validates a Tangled record immediately before it is written
// to a PDS. Generated types supply the wire shape; this checks constraints
// that Go's type system cannot express.
func ValidateRecord(collection string, record any) error {
	switch value := record.(type) {
	case Repo:
		return validateRepo(collection, value)
	case RepoIssue:
		return validateIssue(collection, value)
	case RepoIssueComment:
		return validateIssueComment(collection, value)
	case FeedComment:
		return validateFeedComment(collection, value)
	case RepoIssueState:
		return validateIssueState(collection, value)
	case RepoPull:
		return validatePull(collection, value)
	case RepoPullComment:
		return validatePullComment(collection, value)
	case RepoPullStatus:
		return validatePullStatus(collection, value)
	case PublicKey:
		return validatePublicKey(collection, value)
	case String:
		return validateString(collection, value)
	case map[string]any:
		if collection != "sh.tangled.repo" {
			return fmt.Errorf("%s record has unsupported generated type %T", collection, record)
		}
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode %s record for validation: %w", collection, err)
		}
		var repo Repo
		if err := json.Unmarshal(data, &repo); err != nil {
			return fmt.Errorf("decode %s record for validation: %w", collection, err)
		}
		return validateRepo(collection, repo)
	default:
		return fmt.Errorf("%s record has unsupported generated type %T", collection, record)
	}
}

func validateRepo(collection string, record Repo) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo"); err != nil {
		return err
	}
	if err := required("knot", record.Knot); err != nil {
		return err
	}
	if err := datetime(record.CreatedAt); err != nil {
		return fieldError("createdAt", err)
	}
	if record.Description != nil && !lengthBetween(*record.Description, 1, 140) {
		return fmt.Errorf("description must contain 1 to 140 characters")
	}
	if record.RepoDid != nil {
		if err := did(*record.RepoDid); err != nil {
			return fieldError("repoDid", err)
		}
	}
	for name, value := range map[string]*string{"source": record.Source, "website": record.Website} {
		if value != nil {
			if err := uri(*value); err != nil {
				return fieldError(name, err)
			}
		}
	}
	for _, label := range record.Labels {
		if err := atURI(label); err != nil {
			return fieldError("labels", err)
		}
	}
	if len(record.Topics) > 50 {
		return fmt.Errorf("topics must not contain more than 50 items")
	}
	for _, topic := range record.Topics {
		if !lengthBetween(topic, 1, 50) {
			return fmt.Errorf("topics must contain 1 to 50 characters per item")
		}
	}
	return nil
}

func validateIssue(collection string, record RepoIssue) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo.issue"); err != nil {
		return err
	}
	if err := did(record.Repo); err != nil {
		return fieldError("repo", err)
	}
	if err := required("title", record.Title); err != nil {
		return err
	}
	if err := validateMentions(record.Mentions); err != nil {
		return err
	}
	if err := validateReferences(record.References); err != nil {
		return err
	}
	return datetimeField(record.CreatedAt)
}

func validateIssueComment(collection string, record RepoIssueComment) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo.issue.comment"); err != nil {
		return err
	}
	if err := atURI(record.Issue); err != nil {
		return fieldError("issue", err)
	}
	if err := required("body", record.Body); err != nil {
		return err
	}
	if record.ReplyTo != nil {
		if err := atURI(*record.ReplyTo); err != nil {
			return fieldError("replyTo", err)
		}
	}
	if err := validateMentions(record.Mentions); err != nil {
		return err
	}
	if err := validateReferences(record.References); err != nil {
		return err
	}
	return datetimeField(record.CreatedAt)
}

func validateFeedComment(collection string, record FeedComment) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.feed.comment"); err != nil {
		return err
	}
	if err := validateStrongRef("subject", record.Subject); err != nil {
		return err
	}
	if record.Body == nil || record.Body.MarkupMarkdown == nil {
		return fmt.Errorf("body is required")
	}
	if err := required("body.text", record.Body.MarkupMarkdown.Text); err != nil {
		return err
	}
	if record.ReplyTo != nil {
		if err := validateStrongRef("replyTo", record.ReplyTo); err != nil {
			return err
		}
	}
	if record.Subject != nil {
		subject, err := syntax.ParseATURI(record.Subject.Uri)
		if err != nil {
			return fieldError("subject", fmt.Errorf("must be an at:// URI: %w", err))
		}
		if subject.Collection().String() == "sh.tangled.repo.pull" {
			if record.PullRoundIdx == nil {
				return fmt.Errorf("pullRoundIdx is required for pull subjects")
			}
		}
	}
	if record.PullRoundIdx != nil && *record.PullRoundIdx < 0 {
		return fmt.Errorf("pullRoundIdx must be non-negative")
	}
	return datetimeField(record.CreatedAt)
}

func validateStrongRef(name string, ref *comatproto.RepoStrongRef) error {
	if ref == nil {
		return fmt.Errorf("%s is required", name)
	}
	if err := atURI(ref.Uri); err != nil {
		return fieldError(name+".uri", err)
	}
	if _, err := syntax.ParseCID(ref.Cid); err != nil {
		return fieldError(name+".cid", fmt.Errorf("must be a CID: %w", err))
	}
	return nil
}

func validateIssueState(collection string, record RepoIssueState) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo.issue.state"); err != nil {
		return err
	}
	if err := atURI(record.Issue); err != nil {
		return fieldError("issue", err)
	}
	if record.State != "sh.tangled.repo.issue.state.open" && record.State != "sh.tangled.repo.issue.state.closed" {
		return fmt.Errorf("state must be an issue state value")
	}
	return datetimeField(record.CreatedAt)
}

func validatePull(collection string, record RepoPull) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo.pull"); err != nil {
		return err
	}
	if err := required("title", record.Title); err != nil {
		return err
	}
	if record.Target == nil {
		return fmt.Errorf("target is required")
	}
	if err := did(record.Target.Repo); err != nil {
		return fieldError("target.repo", err)
	}
	if err := required("target.branch", record.Target.Branch); err != nil {
		return err
	}
	if len(record.Rounds) == 0 {
		return fmt.Errorf("rounds is required")
	}
	for _, round := range record.Rounds {
		if round == nil || round.PatchBlob == nil {
			return fmt.Errorf("rounds must contain a patch blob")
		}
		if !round.PatchBlob.Ref.Defined() {
			return fmt.Errorf("rounds.patchBlob must reference a blob")
		}
		if err := datetime(round.CreatedAt); err != nil {
			return fieldError("rounds.createdAt", err)
		}
		if round.PatchBlob.MimeType != "application/gzip" {
			return fmt.Errorf("rounds.patchBlob must have MIME type application/gzip")
		}
	}
	if record.Source != nil {
		if err := required("source.branch", record.Source.Branch); err != nil {
			return err
		}
		if record.Source.Repo != nil {
			if err := did(*record.Source.Repo); err != nil {
				return fieldError("source.repo", err)
			}
		}
	}
	if record.DependentOn != nil {
		if err := atURI(*record.DependentOn); err != nil {
			return fieldError("dependentOn", err)
		}
	}
	if err := validateMentions(record.Mentions); err != nil {
		return err
	}
	if err := validateReferences(record.References); err != nil {
		return err
	}
	return datetimeField(record.CreatedAt)
}

func validatePullComment(collection string, record RepoPullComment) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo.pull.comment"); err != nil {
		return err
	}
	if err := atURI(record.Pull); err != nil {
		return fieldError("pull", err)
	}
	if err := required("body", record.Body); err != nil {
		return err
	}
	if err := validateMentions(record.Mentions); err != nil {
		return err
	}
	if err := validateReferences(record.References); err != nil {
		return err
	}
	return datetimeField(record.CreatedAt)
}

func validatePullStatus(collection string, record RepoPullStatus) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.repo.pull.status"); err != nil {
		return err
	}
	if err := atURI(record.Pull); err != nil {
		return fieldError("pull", err)
	}
	if record.Status != "sh.tangled.repo.pull.status.open" && record.Status != "sh.tangled.repo.pull.status.closed" && record.Status != "sh.tangled.repo.pull.status.merged" {
		return fmt.Errorf("status must be a pull status value")
	}
	return datetimeField(record.CreatedAt)
}

func validatePublicKey(collection string, record PublicKey) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.publicKey"); err != nil {
		return err
	}
	if err := required("key", record.Key); err != nil {
		return err
	}
	if utf8.RuneCountInString(record.Key) > 4096 {
		return fmt.Errorf("key must not exceed 4096 characters")
	}
	if err := required("name", record.Name); err != nil {
		return err
	}
	return datetimeField(record.CreatedAt)
}

func validateString(collection string, record String) error {
	if err := validateType(collection, record.LexiconTypeID, "sh.tangled.string"); err != nil {
		return err
	}
	if !lengthBetween(record.Filename, 1, 140) {
		return fmt.Errorf("filename must contain 1 to 140 characters")
	}
	if utf8.RuneCountInString(record.Description) > 280 {
		return fmt.Errorf("description must not exceed 280 characters")
	}
	if err := required("contents", record.Contents); err != nil {
		return err
	}
	return datetimeField(record.CreatedAt)
}

func validateType(collection, actual, expected string) error {
	if collection != expected {
		return fmt.Errorf("record type %q does not match collection %q", expected, collection)
	}
	if actual != expected {
		return fmt.Errorf("$type must be %q", expected)
	}
	return nil
}

func datetimeField(value string) error {
	if err := datetime(value); err != nil {
		return fieldError("createdAt", err)
	}
	return nil
}

func required(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

func lengthBetween(value string, minimum, maximum int) bool {
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}

func validateMentions(mentions []string) error {
	for _, mention := range mentions {
		if err := did(mention); err != nil {
			return fieldError("mentions", err)
		}
	}
	return nil
}

func validateReferences(references []string) error {
	for _, reference := range references {
		if err := atURI(reference); err != nil {
			return fieldError("references", err)
		}
	}
	return nil
}

func datetime(value string) error {
	if value == "" {
		return fmt.Errorf("is required")
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("must be RFC 3339: %w", err)
	}
	return nil
}

func did(value string) error {
	if _, err := syntax.ParseDID(value); err != nil {
		return fmt.Errorf("must be a DID: %w", err)
	}
	return nil
}

func atURI(value string) error {
	if _, err := syntax.ParseATURI(value); err != nil {
		return fmt.Errorf("must be an at:// URI: %w", err)
	}
	return nil
}

func fieldError(name string, err error) error {
	return fmt.Errorf("%s %w", name, err)
}

func uri(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("must be a URI")
	}
	return nil
}
