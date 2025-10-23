package note

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockConfigService struct {
	storage string
	editor  string
	v_line  bool
}

func (m *mockConfigService) GetStorage() string {
	return m.storage
}

func (m *mockConfigService) GetEditor() string {
	return m.editor
}

func (m *mockConfigService) GetVLineEnabledByDefault() bool {
	return m.v_line
}

func (m *mockConfigService) SetEditor(editor string) error {
	m.editor = editor
	return nil
}

func (m *mockConfigService) SetDefaultVLineStatus(enabled bool) error {
	m.v_line = enabled
	return nil
}

type mockClipboardService struct {
	CopiedText string
}

func (m *mockClipboardService) copy(text string) error {
	m.CopiedText = text
	return nil
}

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	tempDir := t.TempDir()
	mockConfig := &mockConfigService{
		storage: tempDir,
		editor:  "vim",
		v_line:  false,
	}

	store := &Store{
		storage:          tempDir,
		editor:           mockConfig.editor,
		notesDictionary:  make(map[string]Note),
		configService:    mockConfig,
		clipboardService: &mockClipboardService{},
	}
	return store
}

func TestStore_SetEditor(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	newEditor := "neovim"
	err := store.SetEditor(newEditor)

	assert.Equal(t, newEditor, store.editor)
	assert.NoError(t, err)
}

func TestStore_loadNoteFromFile(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	name := "test-note"
	content := "This is a test note"
	filePath := filepath.Join(store.storage, name+".md")

	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)

	note, err := store.loadNoteFromFile(filePath)
	assert.NoError(t, err)

	assert.Equal(t, name, note.Name)
	assert.Equal(t, content, note.Content)
	assert.NotEmpty(t, note.UpdatedAt)
	assert.Equal(t, []byte(content), note.Byte)
}

func TestStore_Create(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	noteName := "new-note"
	noteContent := "Content for the new note."

	err := store.Create(noteName, noteContent)
	assert.NoError(t, err)

	filePath := filepath.Join(store.storage, noteName+".md")
	assert.FileExists(t, filePath)

	data, readErr := os.ReadFile(filePath)
	assert.NoError(t, readErr)
	assert.Equal(t, noteContent, string(data))

	assert.Equal(t, noteName, store.currentNoteName)
}

func TestStore_saveNote(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	note := Note{
		Name:      "test-note",
		Content:   "This is a test note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.saveNote(note.Name, note)
	assert.NoError(t, err)

	filePath := filepath.Join(store.storage, note.Name+".md")
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, note.Content, string(data))
}

func TestStore_saveNote_DirectoryCreation(t *testing.T) {
	t.Parallel()
	tempDir := filepath.Join(t.TempDir(), "nonexistent")

	store := &Store{
		storage:         tempDir,
		editor:          "vim",
		notesDictionary: make(map[string]Note),
	}

	note := Note{
		Name:      "test-note",
		Content:   "This is a test note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.saveNote(note.Name, note)
	assert.NoError(t, err)

	assert.DirExists(t, tempDir)

	filePath := filepath.Join(tempDir, note.Name+".md")
	assert.FileExists(t, filePath)
}

func TestStore_saveNote_NameCollision(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	// First, create the initial note
	firstNote := Note{
		Name:      "test-note",
		Content:   "First note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.saveNote(firstNote.Name, firstNote)
	assert.NoError(t, err)
	store.notesDictionary["test-note"] = firstNote

	// Now create a second note with the same name
	note := Note{
		Name:      "test-note",
		Content:   "This is a test note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate unique name and save
	uniqueName := store.generateUniqueName(note.Name)
	err = store.saveNote(uniqueName, note)
	assert.NoError(t, err)
	assert.Equal(t, "test-note-1", uniqueName)

	filePath := filepath.Join(store.storage, uniqueName+".md")
	assert.FileExists(t, filePath)
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	note := Note{
		Name:      "test-note",
		Content:   "This is a test note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.saveNote(note.Name, note)
	assert.NoError(t, err)

	err = store.Delete(note.Name)
	assert.NoError(t, err)

	filePath := filepath.Join(store.storage, note.Name+".md")
	assert.NoFileExists(t, filePath)
}

func TestStore_Delete_NonExistent(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	err := store.Delete("non-existent-note")

	assert.Error(t, err)
	assert.True(t, os.IsNotExist(errors.Unwrap(err)), "Error should wrap os.ErrNotExist for non-existent file")
}

func TestStore_RenameNote(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	note := Note{
		Name:      "test-note",
		Content:   "This is a test note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.saveNote(note.Name, note)
	assert.NoError(t, err)

	newName := "renamed-note"
	_, err = store.RenameNote(note.Name, newName)
	assert.NoError(t, err)

	filePath := filepath.Join(store.storage, newName+".md")
	assert.FileExists(t, filePath)
}

func TestStore_RenameNote_NonExistent(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)
	newName := "new-name-for-nothing"

	// Attempt to rename a note that doesn't exist
	_, err := store.RenameNote("non-existent-note", newName) //

	// Renaming a non-existent file should also return an error, likely related to the source path.
	assert.Error(t, err)
	// You might want to check for specific error types or messages depending on os.Rename behavior
	assert.Contains(t, err.Error(), "failed to rename note file", "Error message should indicate rename failure")
}

func TestStore_RenameCurrentNote_Collision(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	err := store.Create("note-a", "content a")
	assert.NoError(t, err)
	err = store.Create("note-b", "content b")
	assert.NoError(t, err)

	_, err = store.LoadNotes()
	assert.NoError(t, err)

	store.SetCurrentNoteName("note-b")

	desiredName := "note-a"
	renamedNote, err := store.RenameCurrentNote(desiredName)
	assert.NoError(t, err)

	// Expect it to be renamed to 'note-a-1' because 'note-a' exists
	expectedNewName := "note-a-1"
	assert.Equal(t, expectedNewName, renamedNote.Name)
	assert.Equal(t, expectedNewName, store.currentNoteName, "Current note name should be updated after rename")

	assert.NoFileExists(t, store.GetNotePath("note-b")) //
	assert.FileExists(t, store.GetNotePath("note-a"))
	assert.FileExists(t, store.GetNotePath(expectedNewName))

	_, existsB := store.notesDictionary["note-b"]
	assert.False(t, existsB)
	_, existsA1 := store.notesDictionary[expectedNewName]
	assert.True(t, existsA1)
}

func TestStore_LoadNotes_Multiple(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	noteNames := []string{"note1", "note2", "note3"}

	for i, name := range noteNames {
		time.Sleep(10 * time.Millisecond)

		note := Note{
			Name:      name,
			Content:   fmt.Sprintf("This is test note %d", i+1),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := store.saveNote(note.Name, note)
		assert.NoError(t, err)
	}

	notes, err := store.LoadNotes()
	assert.NoError(t, err)
	assert.Len(t, notes, 3)

	assert.Equal(t, "note3", notes[0].Name)
	assert.Equal(t, "note2", notes[1].Name)
	assert.Equal(t, "note1", notes[2].Name)
}

func TestStore_LoadNotes_Empty(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	notes, err := store.LoadNotes()
	assert.NoError(t, err)
	assert.Empty(t, notes)
}

func TestStore_LoadNotes_MixedFiles(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	mdNote := Note{
		Name:      "markdown-note",
		Content:   "This is a markdown note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.saveNote(mdNote.Name, mdNote)
	assert.NoError(t, err)

	nonMdPath := filepath.Join(store.storage, "not-a-note.txt")
	err = os.WriteFile(nonMdPath, []byte("This is not a markdown file"), 0644)
	assert.NoError(t, err)

	notes, err := store.LoadNotes()
	assert.NoError(t, err)
	assert.Len(t, notes, 1)
	assert.Equal(t, "markdown-note", notes[0].Name)
}

func TestStore_GetCurrentNote(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	_, ok := store.GetCurrentNote()
	assert.False(t, ok, "Should not get a current note when none is set")

	noteName := "current-test"
	noteContent := "This is the current note."
	store.notesDictionary[noteName] = Note{Name: noteName, Content: noteContent}
	store.SetCurrentNoteName(noteName)

	currentNote, ok := store.GetCurrentNote()
	assert.True(t, ok, "Should get the current note")
	assert.Equal(t, noteName, currentNote.Name)
	assert.Equal(t, noteContent, currentNote.Content)

	store.SetCurrentNoteName("non-existent-note")
	_, ok = store.GetCurrentNote()
	assert.False(t, ok, "Should not get a current note for a non-existent name")
}

func TestStore_IsFirstNote(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t)

	assert.True(t, store.IsFirstNote(), "Should be considered 'first' (or irrelevant) when store is empty") //

	err := store.Create("first-note", "content1")
	assert.NoError(t, err)

	_, err = store.LoadNotes()
	assert.NoError(t, err)
	assert.True(t, store.IsFirstNote(), "The only note created should be the first") //

	time.Sleep(10 * time.Millisecond)
	err = store.Create("second-note", "content2")
	assert.NoError(t, err)
	_, err = store.LoadNotes()
	assert.NoError(t, err)

	assert.Equal(t, "second-note", store.currentNoteName, "Second note should be current after create")
	assert.True(t, store.IsFirstNote(), "The latest note created/loaded should be the first") //

	store.SetCurrentNoteName("first-note")
	assert.False(t, store.IsFirstNote(), "Should not be the first note when an older one is selected") //
}
