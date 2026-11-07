package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const testAttestationActor = "did:plc:owner"

// attestationQueue holds files written into one actor's queue directory.
type attestationQueue struct {
	dir     string
	valid   []string // rkeys of well-formed entries
	invalid []string // file names of entries that must be preserved
}

func writeQueueFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func validQueueEntry(t *testing.T, rkey string) []byte {
	t.Helper()
	entry := AttestationEntry{Rkey: rkey, CreatedAt: "2026-11-01T00:00:00Z"}
	entry.Subject.URI = "at://did:plc:repo/" + tangled.IssueCollection + "/" + rkey
	entry.Subject.CID = testCID
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// seedMixedQueue writes two valid entries and three distinct kinds of bad
// entries: unparseable JSON, an entry whose key disagrees with its file name,
// and an entry with a malformed subject.
func seedMixedQueue(t *testing.T, service *Service, actor string) attestationQueue {
	t.Helper()
	dir, err := service.attestationDir(actor)
	if err != nil {
		t.Fatal(err)
	}
	queue := attestationQueue{dir: dir}
	for i := range 2 {
		rkey := string(syntax.NewTID(1_800_000_000_000_000+int64(i), 0))
		writeQueueFile(t, dir, rkey+".json", validQueueEntry(t, rkey))
		queue.valid = append(queue.valid, rkey)
	}
	corrupt := string(syntax.NewTID(1_700_000_000_000_000, 0)) + ".json"
	writeQueueFile(t, dir, corrupt, []byte(`{"rkey":`))
	mismatched := string(syntax.NewTID(1_700_000_000_000_001, 0)) + ".json"
	writeQueueFile(t, dir, mismatched, validQueueEntry(t, queue.valid[0]))
	badSubjectKey := string(syntax.NewTID(1_700_000_000_000_002, 0))
	badSubject := strings.Replace(string(validQueueEntry(t, badSubjectKey)), testCID, "not-a-cid", 1)
	writeQueueFile(t, dir, badSubjectKey+".json", []byte(badSubject))
	queue.invalid = []string{corrupt, mismatched, badSubjectKey + ".json"}
	return queue
}

func assertInvalidEntriesPreserved(t *testing.T, queue attestationQueue) {
	t.Helper()
	for _, name := range queue.invalid {
		if _, err := os.Stat(filepath.Join(queue.dir, name)); err != nil {
			t.Errorf("invalid entry %s was not preserved: %v", name, err)
		}
	}
}

func assertWarningPerInvalidEntry(t *testing.T, warnings []string, queue attestationQueue) {
	t.Helper()
	for _, name := range queue.invalid {
		matches := 0
		for _, warning := range warnings {
			if strings.Contains(warning, name) {
				matches++
			}
		}
		if matches != 1 {
			t.Errorf("want one warning naming %s, got %d in %q", name, matches, warnings)
		}
	}
}

func TestListPendingAttestationsReportsInvalidEntriesWithoutHidingValidOnes(t *testing.T) {
	service, _, _ := migrationService(t)
	queue := seedMixedQueue(t, service, testAttestationActor)

	result, err := service.ListPendingAttestations(context.Background())
	if err != nil {
		t.Fatalf("one bad entry aborted the listing: %v", err)
	}
	var listed []string
	for _, item := range result.Items {
		listed = append(listed, item.Rkey)
	}
	if strings.Join(listed, ",") != strings.Join(queue.valid, ",") {
		t.Fatalf("listed %v, want valid entries %v", listed, queue.valid)
	}
	assertWarningPerInvalidEntry(t, result.Warnings, queue)
	assertInvalidEntriesPreserved(t, queue)

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil || len(decoded.Warnings) != len(queue.invalid) {
		t.Fatalf("JSON output does not carry warnings: %s", encoded)
	}
}

func TestRetryPendingAttestationsAttestsValidEntriesAndKeepsInvalidOnesQueued(t *testing.T) {
	service, pds, _ := migrationService(t)
	queue := seedMixedQueue(t, service, testAttestationActor)

	result, err := service.RetryPendingAttestations(context.Background())
	if err != nil {
		t.Fatalf("one bad entry aborted the retry: %v", err)
	}
	if result.Retried != len(queue.valid) || result.Pending != len(queue.invalid) {
		t.Fatalf("retried=%d pending=%d, want %d and %d", result.Retried, result.Pending, len(queue.valid), len(queue.invalid))
	}
	var putKeys []string
	for _, put := range pds.puts {
		putKeys = append(putKeys, put.Rkey)
	}
	if strings.Join(putKeys, ",") != strings.Join(queue.valid, ",") {
		t.Fatalf("put %v, want only valid entries %v", putKeys, queue.valid)
	}
	assertWarningPerInvalidEntry(t, result.MutationWarnings(), queue)
	assertInvalidEntriesPreserved(t, queue)

	// A second retry must not re-attest finished work and must still surface
	// the entries it cannot process.
	again, err := service.RetryPendingAttestations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again.Retried != 0 || again.Pending != len(queue.invalid) || len(pds.puts) != len(queue.valid) {
		t.Fatalf("second retry=%+v puts=%d", again, len(pds.puts))
	}
	assertWarningPerInvalidEntry(t, again.Warnings, queue)
}

func TestInvalidAttestationEntriesStayInTheirOwnAccountQueue(t *testing.T) {
	service, pds, _ := migrationService(t)
	other := seedMixedQueue(t, service, "did:plc:other")

	listed, err := service.ListPendingAttestations(context.Background())
	if err != nil || len(listed.Items) != 0 || len(listed.Warnings) != 0 {
		t.Fatalf("another account's queue leaked into the active one: %+v err=%v", listed, err)
	}
	retried, err := service.RetryPendingAttestations(context.Background())
	if err != nil || retried.Retried != 0 || retried.Pending != 0 || len(retried.Warnings) != 0 || len(pds.puts) != 0 {
		t.Fatalf("retry touched another account's queue: %+v puts=%d err=%v", retried, len(pds.puts), err)
	}
	for _, rkey := range other.valid {
		if _, err := os.Stat(filepath.Join(other.dir, rkey+".json")); err != nil {
			t.Errorf("other account's entry %s removed: %v", rkey, err)
		}
	}
	assertInvalidEntriesPreserved(t, other)
}

func TestCancelledAttestationRetryCountsUnattemptedEntries(t *testing.T) {
	service, pds, _ := migrationService(t)
	queue := seedMixedQueue(t, service, testAttestationActor)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pds.putHook = func(context.Context) { cancel() }
	result, err := service.RetryPendingAttestations(ctx)
	if !errors.Is(err, context.Canceled) || result.Retried != 1 || len(pds.puts) != 1 {
		t.Fatalf("result=%+v puts=%d err=%v", result, len(pds.puts), err)
	}
	files, err := os.ReadDir(queue.dir)
	if err != nil || result.Pending != len(files) {
		t.Fatalf("pending=%d files=%d err=%v", result.Pending, len(files), err)
	}
}
