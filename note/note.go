package note

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ionut-t/notes/internal/config"
	"github.com/ionut-t/notes/internal/utils"
)

type Note struct {
	Name      string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Byte      []byte
}

type Store struct {
	storage         string
	editor          string
	notes           []Note
	notesDictionary map[string]Note
	currentNoteName string
}

func NewStore() *Store {
	storage := config.GetStorage()
	editor := config.GetEditor()

	store := &Store{
		storage:         storage,
		editor:          editor,
		notesDictionary: make(map[string]Note),
	}

	return store
}

func (s *Store) GetCurrentNote() (Note, bool) {
	if note, ok := s.notesDictionary[s.currentNoteName]; ok {
		return note, true
	}

	return Note{}, false
}

func (s *Store) SetCurrentNoteName(name string) {
	s.currentNoteName = name
}

func (s *Store) Create(name, content string) error {
	note := Note{
		Name:      name,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.saveNote(note)
}

func (s *Store) Update(name, content string) error {
	note, ok := s.GetNote(name)

	if !ok {
		return fmt.Errorf("note %s not found", name)
	}

	note.Content = content
	note.UpdatedAt = time.Now()

	return s.saveNote(note)
}

func (s *Store) DeleteCurrentNote() error {
	if note, ok := s.GetCurrentNote(); ok {
		return s.Delete(note.Name)
	}

	return errors.New("note not found")
}

func (s *Store) Delete(name string) error {
	path := filepath.Join(s.storage, name+".md")

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete note file: %w", err)
	}

	notes := slices.DeleteFunc(s.notes, func(n Note) bool {
		return n.Name == name
	})

	s.notes = notes

	return nil
}

func (s *Store) RenameCurrentNote(newName string) (Note, error) {
	if note, ok := s.GetCurrentNote(); ok {
		return s.RenameNote(note.Name, newName)
	}

	return Note{}, errors.New("note not found")
}

func (s Store) RenameNote(currentName, newName string) (Note, error) {
	currentPath := s.GetNotePath(currentName)

	newName = s.generateUniqueName(newName)

	newPath := s.GetNotePath(newName)

	if err := os.Rename(currentPath, newPath); err != nil {
		return Note{}, fmt.Errorf("failed to rename note file: %w", err)
	}

	for i, note := range s.notes {
		if note.Name == currentName {
			s.notes[i].Name = newName
			delete(s.notesDictionary, currentName)
			s.notesDictionary[newName] = s.notes[i]
			s.currentNoteName = newName
			return s.notes[i], nil
		}
	}

	return Note{}, nil
}

func (s *Store) LoadNotes() ([]Note, error) {
	notes := []Note{}

	err := filepath.WalkDir(s.storage, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}

			return err
		}

		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		note, err := s.loadNoteFromFile(path)
		if err != nil {
			return fmt.Errorf("error loading note %s: %w", path, err)
		}

		notes = append(notes, note)
		s.notesDictionary[note.Name] = note
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking notes directory: %w", err)
	}

	slices.SortStableFunc(notes, func(i, j Note) int {
		if i.UpdatedAt.After(j.UpdatedAt) {
			return -1
		}

		if i.UpdatedAt.Before(j.UpdatedAt) {
			return 1
		}

		return 0
	})

	s.notes = notes

	if len(notes) > 0 {
		s.currentNoteName = utils.Ternary(s.currentNoteName == "", notes[0].Name, s.currentNoteName)
	}

	return notes, nil
}

// used to determine if the note was updated externally
// which means that its position in the list might have changed
func (s *Store) IsFirstNote() bool {
	if len(s.notes) == 0 {
		return true
	}

	return s.currentNoteName == s.notes[0].Name
}

func (s *Store) GetNote(name string) (Note, bool) {
	path := filepath.Join(s.storage, name+".md")
	note, err := s.loadNoteFromFile(path)

	if err == nil {
		return note, true
	}

	notes, err := s.LoadNotes()

	if err != nil {
		return Note{}, false
	}

	for _, n := range notes {
		if strings.Contains(n.Name, name) {
			return n, true
		}
	}

	return Note{}, false
}

func (s *Store) GetNoteByName(name string) (Note, bool) {
	if note, ok := s.notesDictionary[name]; ok {
		return note, true
	}

	return Note{}, false
}

func (s Store) GetEditor() string {
	return s.editor
}

func (s Store) GetStorage() string {
	return s.storage
}

func (s *Store) SetEditor(editor string) error {
	err := config.SetEditor(editor)

	if err != nil {
		return err
	}

	s.editor = editor
	return nil
}

func (s Store) SetDefaultVLineStatus(enabled bool) error {
	return config.SetDefaultVLineStatus(enabled)
}

func (s Store) GetNotePath(name string) string {
	return filepath.Join(s.storage, name+".md")
}

func (s Store) GetVLineEnabledByDefault() bool {
	return config.GetVLineEnabledByDefault()
}

func (s *Store) CopyContent(content string) error {
	if err := clipboard.WriteAll(content); err != nil {
		return fmt.Errorf("failed to copy note content: %w", err)
	}

	return nil
}

func (s *Store) CopyLines(content string, start, end int) (int, error) {
	lines := strings.Split(content, "\n")

	if end == math.MaxInt32 {
		end = len(lines)
	}

	if end > len(lines) {
		end = len(lines)
	}

	start--
	end--

	if len(lines) == 0 {
		return 0, nil
	}

	if start < 0 || start >= len(lines) {
		return 0, fmt.Errorf("invalid start line number %d", start+1)
	}

	if end < 0 || end >= len(lines) {
		return 0, fmt.Errorf("invalid end line number %d", end+1)
	}

	if start > end {
		return 0, fmt.Errorf("invalid range: start line (%d) is greater than end line (%d)", start+1, end+1)
	}

	content = strings.Join(lines[start:end+1], "\n")

	if err := clipboard.WriteAll(content); err != nil {
		return 0, fmt.Errorf("failed to copy note content: %w", err)
	}

	copiedLines := end - start + 1

	return copiedLines, nil
}

// Handles various formats:
// - co 1 2 (copy lines 1 to 2)
// - co 1 1 (copy line 1)
// - co 1 (copy line 1)
// - co 20 > 2 (copy lines 20 to 22)
// - co 20 < 2 (copy lines 18 to 20)
// - co 20 > -1 (copy lines 20 to the end)
// - co 20 < -1 (copy lines 1 to 20)
func ParseCopyLinesCommand(cmd string) (int, int, error) {
	// Define the regexes for different command patterns
	copyBasicRe := regexp.MustCompile(`^co\s+(\d+)(?:\s+(\d+))?$`)
	copyRelativeRe := regexp.MustCompile(`^co\s+(\d+)\s+([<>])\s+(-?\d+)$`)

	// Check if it's a basic pattern (co NUM [NUM])
	if matches := copyBasicRe.FindStringSubmatch(cmd); len(matches) >= 2 {
		start, startErr := strconv.Atoi(matches[1])
		if startErr != nil {
			return 0, 0, fmt.Errorf("invalid start line: %v", startErr)
		}

		// If only one number provided, start and end are the same
		end := start
		if len(matches) == 3 && matches[2] != "" {
			var endErr error
			end, endErr = strconv.Atoi(matches[2])
			if endErr != nil {
				return 0, 0, fmt.Errorf("invalid end line: %v", endErr)
			}
		}

		return start, end, nil
	}

	// Check if it's a relative pattern (co NUM < NUM or co NUM > NUM)
	if matches := copyRelativeRe.FindStringSubmatch(cmd); len(matches) == 4 {
		base, baseErr := strconv.Atoi(matches[1])
		if baseErr != nil {
			return 0, 0, fmt.Errorf("invalid base line: %v", baseErr)
		}

		op := matches[2] // < or >

		offset, offsetErr := strconv.Atoi(matches[3])
		if offsetErr != nil {
			return 0, 0, fmt.Errorf("invalid offset: %v", offsetErr)
		}

		var start, end int

		if op == ">" {
			// co 20 > 2 means lines 20 to (20+2)
			start = base

			if offset == -1 {
				// Special case: co 20 > -1 means copy from line 20 to the end
				// Instead of using -1, we need to calculate the actual end line
				// This will be handled in the calling code by setting end to last line
				end = math.MaxInt32 // Very large number to be clamped by caller
			} else {
				end = base + offset
			}
		} else { // op == "<"
			// co 20 < 2 means lines (20-2) to 20

			if offset == -1 {
				// Special case: co 20 < -1 means copy from line 1 to line 20
				start = 1
			} else {
				start = base - offset
				start = max(1, start) // Ensure start is at least 1
			}

			end = base
		}

		return start, end, nil
	}

	return 0, 0, fmt.Errorf("invalid command format: %s", cmd)
}

func (s Store) getAllNoteFileNames() []string {
	var notes []Note

	if loadedNotes, err := s.LoadNotes(); err == nil {
		notes = loadedNotes
	} else {
		notes = s.notes
	}

	fileNames := []string{}
	for _, note := range notes {
		name := strings.ToLower(strings.TrimSuffix(note.Name, ".md"))
		fileNames = append(fileNames, name)
	}

	return fileNames
}

// SaveNote saves a note to the store
func (s *Store) saveNote(note Note) error {
	path := filepath.Join(s.storage, s.generateUniqueName(note.Name)+".md")

	// Create the note content
	content := strings.Trim(note.Content, "\n")

	// check if directory exists
	if _, err := os.Stat(s.storage); os.IsNotExist(err) {
		if err := os.MkdirAll(s.storage, 0755); err != nil {
			return fmt.Errorf("failed to create notes directory: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write note file: %w", err)
	}

	return nil
}

func (s Store) generateUniqueName(name string) string {
	noteFileNames := s.getAllNoteFileNames()

	if len(noteFileNames) == 0 {
		return name
	}

	originalName := name
	counter := 1

	for {
		duplicate := false
		for _, n := range noteFileNames {
			if n == strings.ToLower(name) {
				duplicate = true
				break
			}
		}

		if !duplicate {
			break
		}

		name = originalName + "-" + strconv.Itoa(counter)
		counter++
	}

	return name
}

func getCreationTime(filePath string) (time.Time, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, fmt.Errorf("failed to get raw syscall.Stat_t")
	}

	// Different OS has different fields in syscall.Stat_t
	switch runtime.GOOS {
	case "darwin":
		// macOS has Birthtimespec for creation time
		return time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec)), nil
	case "windows":
		// Windows implementation would be different and should use syscall.GetFileTime
		return time.Time{}, fmt.Errorf("use separate Windows implementation")
	default:
		// Linux generally doesn't store true creation time, using Ctim (status change time)
		return time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec)), nil
	}
}

func (s *Store) loadNoteFromFile(path string) (Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Note{}, fmt.Errorf("failed to read note file: %w", err)
	}

	content := string(data)

	name := strings.TrimSuffix(filepath.Base(path), ".md")

	fileInfo, err := os.Stat(path)

	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return Note{}, err
	}

	createdAt, err := getCreationTime(path)

	if err != nil {
		fmt.Printf("Error getting file creation time: %v\n", err)
		return Note{}, err
	}

	updatedAt := fileInfo.ModTime()

	return Note{
		Name:      name,
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Byte:      data,
	}, nil
}
