package atproto

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

func newTestInsecureFileStore(t *testing.T) *insecureFileStore {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	store, err := newInsecureFileStore()
	if err != nil {
		t.Fatalf("newInsecureFileStore: %v", err)
	}
	return store
}

func TestFileStore_RoundTrip(t *testing.T) {
	store := newTestInsecureFileStore(t)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	session := samplePasswordSession(did, "https://pds.example.com")

	if err := store.SavePasswordSession("alice.example.com", session); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	got, handle, err := store.GetPasswordSession()
	if err != nil {
		t.Fatalf("GetPasswordSession: %v", err)
	}
	if handle != "alice.example.com" {
		t.Errorf("handle = %q, want %q", handle, "alice.example.com")
	}
	if got.AccountDID != did {
		t.Errorf("AccountDID = %v, want %v", got.AccountDID, did)
	}
	if got.AccessToken != "access" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "access")
	}
}

func TestFileStore_NotFoundWhenAbsent(t *testing.T) {
	store := newTestInsecureFileStore(t)
	_, _, err := store.GetPasswordSession()
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("expected keyring.ErrNotFound, got %v", err)
	}
}

func TestFileStore_DeleteIsIdempotent(t *testing.T) {
	store := newTestInsecureFileStore(t)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := store.SavePasswordSession("alice.example.com", samplePasswordSession(did, "https://pds.example.com")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}
	if err := store.DeletePasswordSession(); err != nil {
		t.Fatalf("first DeletePasswordSession: %v", err)
	}
	if err := store.DeletePasswordSession(); err != nil {
		t.Fatalf("second DeletePasswordSession: %v", err)
	}
}

func TestFileStore_FilePermissions(t *testing.T) {
	store := newTestInsecureFileStore(t)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := store.SavePasswordSession("alice.example.com", samplePasswordSession(did, "https://pds.example.com")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	info, err := os.Stat(store.path)
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 0600", perm)
	}

	dir := filepath.Dir(store.path)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat credentials dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir mode = %o, want 0700", perm)
	}
}

func TestFileStore_TightensExistingDirectoryPermissions(t *testing.T) {
	dataHome := t.TempDir()
	directory := filepath.Join(dataHome, "tg")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", dataHome)
	store, err := newInsecureFileStore()
	if err != nil {
		t.Fatalf("newInsecureFileStore() error = %v", err)
	}
	info, err := os.Stat(filepath.Dir(store.path))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("credentials directory mode = %o", info.Mode().Perm())
	}
}

func TestFileStore_OverwritesPreviousEntry(t *testing.T) {
	store := newTestInsecureFileStore(t)
	first := mustDID(t, "did:plc:111111111111111111111111")
	second := mustDID(t, "did:plc:222222222222222222222222")

	if err := store.SavePasswordSession("alice.example.com", samplePasswordSession(first, "https://pds.example.com")); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := store.SavePasswordSession("bob.example.com", samplePasswordSession(second, "https://pds.example.com")); err != nil {
		t.Fatalf("save second: %v", err)
	}

	got, handle, err := store.GetPasswordSession()
	if err != nil {
		t.Fatalf("GetPasswordSession: %v", err)
	}
	if got.AccountDID != second {
		t.Errorf("AccountDID = %v, want %v (latest entry)", got.AccountDID, second)
	}
	if handle != "bob.example.com" {
		t.Errorf("handle = %q, want %q", handle, "bob.example.com")
	}
}

// newAuthManagerWithFileStore builds an AuthManager backed by a file store
// rooted in a temp directory and a fake keyring (so keyring lookups always miss
// unless a test seeds them).
func newAuthManagerWithFileStore(t *testing.T, callbackURL string) *AuthManager {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	insecureStore, err := newInsecureFileStore()
	if err != nil {
		t.Fatalf("newInsecureFileStore: %v", err)
	}
	manager := newAuthManagerForTest(callbackURL, testKeyringStore(newFakeKeyring()))
	manager.insecureStore = insecureStore
	return manager
}

func saveFileSession(t *testing.T, manager *AuthManager, did syntax.DID) {
	t.Helper()
	if err := manager.insecureStore.SavePasswordSession("alice.example.com", samplePasswordSession(did, "https://pds.example.com")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}
}

func makeKeyringUnavailable(t *testing.T, manager *AuthManager) {
	t.Helper()
	backend, ok := manager.store.backend.(*fakeKeyring)
	if !ok {
		t.Fatal("test manager does not use a fake keyring")
	}
	backend.getErr = errors.New("Secret Service is unavailable")
}

func TestAuthManager_CurrentDIDFromFile(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)

	got, err := manager.CurrentDID(context.Background())
	if err != nil {
		t.Fatalf("CurrentDID: %v", err)
	}
	if got != did {
		t.Errorf("CurrentDID = %v, want %v", got, did)
	}
}

func TestAuthManager_CurrentDIDFromFileWhenKeyringUnavailable(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)
	makeKeyringUnavailable(t, manager)

	got, err := manager.CurrentDID(context.Background())
	if err != nil {
		t.Fatalf("CurrentDID: %v", err)
	}
	if got != did {
		t.Errorf("CurrentDID = %v, want %v", got, did)
	}
}

func TestAuthManager_PrefersKeyringAccountOverFileAccount(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	keyringDID := mustDID(t, "did:plc:111111111111111111111111")
	fileDID := mustDID(t, "did:plc:222222222222222222222222")
	if err := manager.store.SavePasswordSession(context.Background(), samplePasswordSession(keyringDID, "https://pds.example.com")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}
	saveFileSession(t, manager, fileDID)

	account, source, err := manager.activeAccount()
	if err != nil {
		t.Fatalf("activeAccount: %v", err)
	}
	if source != sessionSourceKeyring {
		t.Errorf("source = %v, want keyring", source)
	}
	if account.DID != keyringDID.String() {
		t.Errorf("account DID = %q, want %q", account.DID, keyringDID)
	}
}

func TestAuthManager_CurrentSessionFileIsNotOAuth(t *testing.T) {
	// The file store is password-only, so CurrentSession must report
	// ErrNotAuthenticated so callers (e.g. AccessToken) fall back to APIClient.
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)
	_, err := manager.CurrentSession(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated for file-backed session, got %v", err)
	}
}

func TestAuthManager_APIClientReadsFromFile(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)

	client, gotDID, err := manager.APIClient(context.Background())
	if err != nil {
		t.Fatalf("APIClient: %v", err)
	}
	if gotDID != did {
		t.Errorf("APIClient DID = %v, want %v", gotDID, did)
	}
	if client == nil {
		t.Fatal("APIClient returned nil client")
	}
}

func TestAuthManager_LogoutDeletesFile(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)

	if err := manager.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, _, err := manager.insecureStore.GetPasswordSession(); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("expected credentials file deleted, got %v", err)
	}
}

func TestAuthManager_LogoutWithoutAnySession(t *testing.T) {
	// Neither keyring nor file store has a session.
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	err := manager.Logout(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestAuthManager_AccountsIncludesFileAccount(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)

	accounts, activeDID, err := manager.Accounts()
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].DID != did.String() {
		t.Errorf("account DID = %q, want %q", accounts[0].DID, did.String())
	}
	if accounts[0].Method != AuthMethodPassword {
		t.Errorf("account method = %q, want %q", accounts[0].Method, AuthMethodPassword)
	}
	if activeDID != did.String() {
		t.Errorf("activeDID = %q, want %q", activeDID, did.String())
	}
}

func TestAuthManager_SelectAccountFindsFileAccount(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)

	account, err := manager.SelectAccount(did.String())
	if err != nil {
		t.Fatalf("SelectAccount by DID: %v", err)
	}
	if account.DID != did.String() {
		t.Errorf("account DID = %q, want %q", account.DID, did.String())
	}

	account, err = manager.SelectAccount("alice.example.com")
	if err != nil {
		t.Fatalf("SelectAccount by handle: %v", err)
	}
	if account.DID != did.String() {
		t.Errorf("account DID = %q, want %q", account.DID, did.String())
	}
}

func TestAuthManager_SelectAccountFindsFileAccountWhenKeyringUnavailable(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)
	makeKeyringUnavailable(t, manager)

	account, err := manager.SelectAccount("alice.example.com")
	if err != nil {
		t.Fatalf("SelectAccount: %v", err)
	}
	if account.DID != did.String() {
		t.Errorf("account DID = %q, want %q", account.DID, did.String())
	}
}

func TestAuthManager_LogoutAllClearsFileAndKeyring(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	fileDID := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, fileDID)

	if err := manager.LogoutAll(context.Background()); err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if _, _, err := manager.insecureStore.GetPasswordSession(); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("expected credentials file deleted after LogoutAll, got %v", err)
	}
}

func TestAuthManager_LogoutAllClearsFileWhenKeyringUnavailable(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	saveFileSession(t, manager, did)
	makeKeyringUnavailable(t, manager)

	if err := manager.LogoutAll(context.Background()); err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if _, _, err := manager.insecureStore.GetPasswordSession(); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("expected credentials file deleted after LogoutAll, got %v", err)
	}
}

func TestAuthManager_LogoutAllWithoutAnySession(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	err := manager.LogoutAll(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestAccountSelectionPersistsAcrossStoresAndRestart(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:8095/callback")
	fileDID := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	keyringDID := mustDID(t, "did:plc:111111111111111111111111")
	saveFileSession(t, manager, fileDID)
	if err := manager.store.SavePasswordSession(context.Background(), samplePasswordSession(keyringDID, "https://pds.example")); err != nil {
		t.Fatal(err)
	}
	for _, did := range []syntax.DID{fileDID, keyringDID, fileDID} {
		if _, err := manager.SelectAccount(did.String()); err != nil {
			t.Fatal(err)
		}
		restored := newAuthManagerForTest("", manager.store)
		restored.insecureStore = &insecureFileStore{path: manager.insecureStore.path}
		got, err := restored.CurrentDID(context.Background())
		if err != nil || got != did {
			t.Fatalf("selected %s, restored %s: %v", did, got, err)
		}
		_, active, err := restored.Accounts()
		if err != nil || active != did.String() {
			t.Fatalf("active=%s err=%v", active, err)
		}
		// Session refresh must not reset the selected store.
		session, handle, err := restored.insecureStore.GetPasswordSession()
		if err != nil {
			t.Fatal(err)
		}
		if err := restored.insecureStore.SavePasswordSession(handle, *session); err != nil {
			t.Fatal(err)
		}
		got, err = restored.CurrentDID(context.Background())
		if err != nil || got != did {
			t.Fatalf("refresh changed selected account from %s to %s: %v", did, got, err)
		}
		manager = restored
	}
}

func TestCorruptOptionalCredentialsDoNotBlockKeyringRecovery(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "")
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := manager.store.SavePasswordSession(context.Background(), samplePasswordSession(did, "https://pds.example")); err != nil {
		t.Fatal(err)
	}
	damaged := []byte("{")
	if err := os.WriteFile(manager.insecureStore.path, damaged, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := manager.CurrentDID(context.Background())
	if err != nil || got != did {
		t.Fatalf("implicit account=%s err=%v", got, err)
	}
	if _, err := manager.SelectAccount(did.String()); err != nil {
		t.Fatal(err)
	}
	restored := newAuthManagerForTest("", manager.store)
	restored.insecureStore = &insecureFileStore{path: manager.insecureStore.path}
	got, err = restored.CurrentDID(context.Background())
	if err != nil || got != did {
		t.Fatalf("restored account=%s err=%v", got, err)
	}
	if _, err := restored.SelectAccount("did:plc:111111111111111111111111"); err == nil || !strings.Contains(err.Error(), manager.insecureStore.path) {
		t.Fatalf("missing credential-file diagnostic: %v", err)
	}
	restored.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	})}
	if err := restored.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.CurrentDID(context.Background()); err == nil {
		t.Fatal("corrupt file was accepted as a session")
	}
	unchanged, err := os.ReadFile(manager.insecureStore.path)
	if err != nil || string(unchanged) != string(damaged) {
		t.Fatalf("optional credentials modified: %q err=%v", unchanged, err)
	}
}

func TestFreshCredentialsReplaceCorruptFileAndCanBeSelected(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "")
	keyringDID := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	fileDID := mustDID(t, "did:plc:111111111111111111111111")
	if err := manager.store.SavePasswordSession(context.Background(), samplePasswordSession(keyringDID, "https://pds.example")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manager.insecureStore.path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.insecureStore.SavePasswordSession("file.example", samplePasswordSession(fileDID, "https://pds.example")); err != nil {
		t.Fatal(err)
	}
	session, _, err := manager.insecureStore.GetPasswordSession()
	if err != nil || session.AccountDID != fileDID {
		t.Fatalf("session=%+v err=%v", session, err)
	}
	got, err := manager.CurrentDID(context.Background())
	if err != nil || got != keyringDID {
		t.Fatalf("saving credentials changed selection: %s err=%v", got, err)
	}
	if _, err := manager.SelectAccount(fileDID.String()); err != nil {
		t.Fatal(err)
	}
	restored := newAuthManagerForTest("", manager.store)
	restored.insecureStore = &insecureFileStore{path: manager.insecureStore.path}
	got, err = restored.CurrentDID(context.Background())
	if err != nil || got != fileDID {
		t.Fatalf("selected replacement not restored: %s err=%v", got, err)
	}
}

func TestFreshCredentialsPreservePermissionErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file read permissions")
	}
	store := newTestInsecureFileStore(t)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := os.WriteFile(store.path, []byte("protected"), 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(store.path, 0o600)
	err := store.SavePasswordSession("file.example", samplePasswordSession(did, "https://pds.example"))
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("err=%v", err)
	}
	if err := os.Chmod(store.path, 0o600); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(store.path)
	if err != nil || string(contents) != "protected" {
		t.Fatalf("protected file replaced: %q err=%v", contents, err)
	}
}
