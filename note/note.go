package note

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
)

type Note struct {
	Name      string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Byte      []byte
}

type NotesStore struct {
	Dir string
}

func NewNotesStore(dir string) (*NotesStore, error) {
	if dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(homeDir, ".notes")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	store := &NotesStore{
		Dir: dir,
	}

	return store, nil
}

func (s *NotesStore) Create(name, content string) error {
	note := Note{
		Name:      name,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.saveNote(note)
}

func (s *NotesStore) Update(name, content string) error {
	note, ok := s.GetNote(name)

	if !ok {
		return fmt.Errorf("note %s not found", name)
	}

	note.Content = content
	note.UpdatedAt = time.Now()

	return s.saveNote(note)
}

func (s *NotesStore) Delete(name string) error {
	path := filepath.Join(s.Dir, name+".md")

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete note file: %w", err)
	}

	return nil
}

// SaveNote saves a note to the store
func (s *NotesStore) saveNote(note Note) error {
	path := filepath.Join(s.Dir, s.generateUniqueFileName(note.Name)+".md")

	// Create the note content
	content := note.Content

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write note file: %w", err)
	}

	return nil
}

func (s NotesStore) generateUniqueFileName(name string) string {
	noteFileNames, err := s.GetAllNoteFileNames()

	if err != nil {
		return name
	}

	originalName := name
	counter := 1

	for {
		duplicate := false
		for _, n := range noteFileNames {
			if n == name {
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

// GetAllNotes retrieves all notes from the store
func (s *NotesStore) GetAllNotes() ([]Note, error) {
	notes := []Note{}

	err := filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
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
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking notes directory: %w", err)
	}

	return notes, nil
}

func (s *NotesStore) GetNote(name string) (Note, bool) {
	path := filepath.Join(s.Dir, name+".md")
	note, err := s.loadNoteFromFile(path)

	if err == nil {
		return note, true
	}

	notes, err := s.GetAllNotes()

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

func (s *NotesStore) loadNoteFromFile(path string) (Note, error) {
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

func (s NotesStore) GetAllNoteFileNames() ([]string, error) {
	fileNames := []string{}

	err := filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		fileNames = append(fileNames, strings.TrimSuffix(d.Name(), ".md"))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking notes directory: %w", err)
	}

	return fileNames, nil
}

func (s NotesStore) GetEditor() string {
	editor := os.Getenv("EDITOR")

	if editor == "" {
		editor = "vim"
	}

	return editor
}

func (s NotesStore) GetNotePath(name string) string {
	return filepath.Join(s.Dir, name+".md")
}

func (s NotesStore) RenameNote(currentName, newName string) (string, error) {
	currentPath := s.GetNotePath(currentName)

	fileName := s.generateUniqueFileName(newName)

	newPath := s.GetNotePath(fileName)

	if err := os.Rename(currentPath, newPath); err != nil {
		return fileName, fmt.Errorf("failed to rename note file: %w", err)
	}

	return fileName, nil
}

func (s *NotesStore) CopyContent(content string) error {
	if err := clipboard.WriteAll(content); err != nil {
		return fmt.Errorf("failed to copy note content: %w", err)
	}

	return nil
}

func (s *NotesStore) CopyLines(content string, start, end int) error {
	lines := strings.Split(content, "\n")

	if len(lines) == 0 {
		return nil
	}

	if start < 0 || start >= len(lines) {
		return fmt.Errorf("invalid start line number")
	}

	if end < 0 || end >= len(lines) {
		return fmt.Errorf("invalid end line number")
	}

	content = strings.Join(lines[start:end+1], "\n")

	if err := clipboard.WriteAll(content); err != nil {
		return fmt.Errorf("failed to copy note content: %w", err)
	}

	return nil
}
func (s *NotesStore) CopyFromCodeBlock(content string, codeBlockNo, codeLine int) error {
	lines := strings.Split(content, "\n")
	codeBlockCount := 0
	startIndex := -1
	endIndex := -1

	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			codeBlockCount++
			if codeBlockCount == codeBlockNo {
				startIndex = i + 1
			} else if startIndex != -1 {
				endIndex = i
				break
			}
		}
	}

	if startIndex == -1 {
		return fmt.Errorf("code block %d not found", codeBlockNo)
	}

	if codeLine == 0 {
		if endIndex == -1 {
			endIndex = len(lines)
		}
		codeBlockContent := strings.Join(lines[startIndex:endIndex], "\n")
		if err := clipboard.WriteAll(codeBlockContent); err != nil {
			return fmt.Errorf("failed to copy code block content: %w", err)
		}
		return nil
	}

	if codeLine < 1 || startIndex+codeLine-1 >= len(lines) || strings.HasPrefix(lines[startIndex+codeLine-1], "```") {
		return fmt.Errorf("code line %d not found in code block %d", codeLine, codeBlockNo)
	}

	codeLineContent := lines[startIndex+codeLine-1]

	if err := clipboard.WriteAll(codeLineContent); err != nil {
		return fmt.Errorf("failed to copy code line content: %w", err)
	}

	return nil
}
