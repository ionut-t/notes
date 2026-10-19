package note

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startWatcher(t *testing.T, store *Store) *Watcher {
	t.Helper()

	w, err := store.Watch()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, w.Close()) })

	return w
}

func waitForChange(t *testing.T, w *Watcher) {
	t.Helper()

	select {
	case <-w.Changes():
	case <-time.After(2 * time.Second):
		t.Fatal("expected a change notification")
	}
}

func TestWatcher_NotifiesOnExternalChanges(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	w := startWatcher(t, store)

	path := filepath.Join(store.storage, "external.md")

	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))
	waitForChange(t, w)

	require.NoError(t, os.WriteFile(path, []byte("hello again"), 0644))
	waitForChange(t, w)

	require.NoError(t, os.Remove(path))
	waitForChange(t, w)
}

func TestWatcher_DebouncesBursts(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	w := startWatcher(t, store)

	for i := range 5 {
		path := filepath.Join(store.storage, "note"+string(rune('a'+i))+".md")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	}

	waitForChange(t, w)

	select {
	case <-w.Changes():
		t.Fatal("expected a single notification for a burst of changes")
	case <-time.After(3 * watcherDebounce):
	}
}

func TestWatcher_IgnoresNonMarkdownFiles(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	w := startWatcher(t, store)

	require.NoError(t, os.WriteFile(filepath.Join(store.storage, ".note.md.swp"), []byte("x"), 0644))

	select {
	case <-w.Changes():
		t.Fatal("expected no notification for non-markdown files")
	case <-time.After(3 * watcherDebounce):
	}
}

func TestWatcher_WatchesNewSubdirectories(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	w := startWatcher(t, store)

	dir := filepath.Join(store.storage, "nested")
	require.NoError(t, os.Mkdir(dir, 0755))
	waitForChange(t, w)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "deep.md"), []byte("x"), 0644))
	waitForChange(t, w)
}

func TestStore_LoadNotes_DropsExternallyDeletedNotes(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	require.NoError(t, store.Create("first", "1"))
	require.NoError(t, store.Create("second", "2"))
	_, err := store.LoadNotes()
	require.NoError(t, err)

	store.SetCurrentNoteName("second")
	require.NoError(t, os.Remove(store.GetNotePath("second")))

	notes, err := store.LoadNotes()
	require.NoError(t, err)
	assert.Len(t, notes, 1)

	current, ok := store.GetCurrentNote()
	assert.True(t, ok)
	assert.Equal(t, "first", current.Name)
}

func TestStore_ReadNotes_DoesNotModifyStore(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	require.NoError(t, store.Create("first", "1"))
	_, err := store.LoadNotes()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(store.GetNotePath("second"), []byte("2"), 0644))

	notes, err := store.ReadNotes()
	require.NoError(t, err)
	assert.Len(t, notes, 2)
	assert.Len(t, store.GetNotes(), 1)

	store.SetNotes(notes)
	assert.Len(t, store.GetNotes(), 2)
}

func TestWatcher_NotifiesOnSaveViaTemporaryFile(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	path := store.GetNotePath("note")
	require.NoError(t, os.WriteFile(path, []byte("v1"), 0644))

	w := startWatcher(t, store)

	for _, tmp := range []string{"note.md~", "note.md.tmp", ".note.md.swp"} {
		tmpPath := filepath.Join(store.storage, tmp)
		require.NoError(t, os.WriteFile(tmpPath, []byte("v2"), 0644))
		require.NoError(t, os.Rename(tmpPath, path))
		waitForChange(t, w)
	}
}

func TestWatcher_IgnoresEditorLockFiles(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	w := startWatcher(t, store)

	require.NoError(t, os.Symlink("user@host.1:1", filepath.Join(store.storage, ".#note.md")))

	select {
	case <-w.Changes():
		t.Fatal("expected no notification for editor lock files")
	case <-time.After(3 * watcherDebounce):
	}
}

func TestStore_ReadNotes_SkipsLockAndDanglingFiles(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	require.NoError(t, os.WriteFile(store.GetNotePath("note"), []byte("1"), 0644))
	require.NoError(t, os.Symlink("user@host.1:1", filepath.Join(store.storage, ".#note.md")))
	require.NoError(t, os.Symlink("missing", filepath.Join(store.storage, "dangling.md")))

	notes, err := store.ReadNotes()
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "note", notes[0].Name)
}
