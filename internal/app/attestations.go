package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/fsutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func nowString() string { return time.Now().UTC().Format(time.RFC3339) }

type AttestationEntry struct {
	Rkey    string `json:"rkey"`
	Subject struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	} `json:"subject"`
	Change    string `json:"change,omitempty"`
	CreatedAt string `json:"createdAt"`
}
type AttestationListResult struct {
	Items    []AttestationEntry `json:"items"`
	Warnings []string           `json:"warnings,omitempty"`
}
type AttestationRetryResult struct {
	Retried  int      `json:"retried"`
	Pending  int      `json:"pending"`
	Warnings []string `json:"warnings,omitempty"`
}

func (s *Service) attestationDir(actor string) (string, error) {
	root := s.attestationDirectory
	if root == "" {
		root = os.Getenv("XDG_STATE_HOME")
		if root == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			root = filepath.Join(home, ".local", "state")
		}
		root = filepath.Join(root, "tg", "attestations")
	}
	digest := sha256.Sum256([]byte(actor))
	return filepath.Join(root, hex.EncodeToString(digest[:])), nil
}
func (s *Service) queueAttestation(ctx context.Context, pds pdsClient, actor string, receipt knot.Receipt) error {
	dir, err := s.attestationDir(actor)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	var clock [2]byte
	if _, err := rand.Read(clock[:]); err != nil {
		return err
	}
	entry := AttestationEntry{Rkey: string(syntax.NewTIDNow(uint(clock[0])<<2 | uint(clock[1]&3))), Change: receipt.Change, CreatedAt: nowString()}
	entry.Subject.URI = receipt.URI
	entry.Subject.CID = receipt.CID
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".pending-")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	path := filepath.Join(dir, entry.Rkey+".json")
	if err = os.Link(temporary, path); err != nil {
		return err
	}
	if err := fsutil.SyncDir(dir); err != nil {
		return err
	}
	return attestEntry(ctx, pds, actor, entry, path)
}
func attestEntry(ctx context.Context, pds pdsClient, actor string, entry AttestationEntry, path string) error {
	record := map[string]any{"$type": tangled.AttestationCollection, "subject": entry.Subject, "createdAt": entry.CreatedAt}
	if entry.Change != "" {
		record["change"] = entry.Change
	}
	if _, _, err := pds.PutRecord(ctx, atproto.PutRecordInput{Repo: actor, Collection: tangled.AttestationCollection, Rkey: entry.Rkey, Record: record}); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// pendingAttestations reads the actor's queue. An entry that cannot be read
// or validated is left on disk and reported as a warning so it neither blocks
// the valid entries nor disappears from view.
func (s *Service) pendingAttestations(actor string) (items []AttestationEntry, dir string, warnings []string, err error) {
	dir, err = s.attestationDir(actor)
	if err != nil {
		return nil, "", nil, err
	}
	files, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []AttestationEntry{}, dir, nil, nil
	}
	if err != nil {
		return nil, "", nil, err
	}
	items = make([]AttestationEntry, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, file.Name())
		entry, err := readPendingAttestation(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped unreadable pending attestation %s (left in place): %v", path, err))
			continue
		}
		items = append(items, entry)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Rkey < items[j].Rkey })
	return items, dir, warnings, nil
}
func readPendingAttestation(path string) (AttestationEntry, error) {
	var entry AttestationEntry
	data, err := os.ReadFile(path)
	if err != nil {
		return entry, err
	}
	if err := json.Unmarshal(data, &entry); err != nil {
		return entry, err
	}
	if entry.Rkey+".json" != filepath.Base(path) {
		return entry, fmt.Errorf("record key %q does not match file name", entry.Rkey)
	}
	if _, err := syntax.ParseTID(entry.Rkey); err != nil {
		return entry, err
	}
	if _, err := syntax.ParseATURI(entry.Subject.URI); err != nil {
		return entry, err
	}
	if _, err := syntax.ParseCID(entry.Subject.CID); err != nil {
		return entry, err
	}
	return entry, nil
}
func (s *Service) ListPendingAttestations(ctx context.Context) (*AttestationListResult, error) {
	actor, err := s.sessions.CurrentDID(ctx)
	if err != nil {
		return nil, err
	}
	items, _, warnings, err := s.pendingAttestations(actor)
	if err != nil {
		return nil, err
	}
	return &AttestationListResult{Items: items, Warnings: warnings}, nil
}

// RetryPendingAttestations re-puts every valid queued attestation. Pending
// counts everything still queued afterwards, including entries that could not
// be read.
func (s *Service) RetryPendingAttestations(ctx context.Context) (*AttestationRetryResult, error) {
	pds, actor, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	items, dir, warnings, err := s.pendingAttestations(actor)
	if err != nil {
		return nil, err
	}
	result := &AttestationRetryResult{Pending: len(warnings) + len(items), Warnings: warnings}
	for _, entry := range items {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := attestEntry(ctx, pds, actor, entry, filepath.Join(dir, entry.Rkey+".json")); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", entry.Subject.URI, err))
		} else {
			result.Pending--
			result.Retried++
		}
	}
	return result, nil
}

func (r *AttestationListResult) MutationWarnings() []string  { return r.Warnings }
func (r *AttestationRetryResult) MutationWarnings() []string { return r.Warnings }
