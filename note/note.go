package note

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ionut-t/notes/internal/config"
	"github.com/ionut-t/notes/internal/utils"
)

type configService interface {
	GetStorage() string
	GetEditor() string
	SetEditor(editor string) error
}

type configServiceImpl struct{}

func (c configServiceImpl) GetStorage() string {
	return config.GetStorage()
}
func (c configServiceImpl) GetEditor() string {
	return config.GetEditor()
}
func (c configServiceImpl) SetEditor(editor string) error {
	return config.SetEditor(editor)
}

type clipboardService interface {
	copy(text string) error
}

type clipboardServiceImpl struct{}

func (c clipboardServiceImpl) copy(text string) error {
	return clipboard.WriteAll(text)
}

type Note struct {
	Name      string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Byte      []byte
}

type Store struct {
	storage          string
	editor           string
	notes            []Note
	notesDictionary  map[string]Note
	currentNoteName  string
	configService    configService
	clipboardService clipboardService
}

func NewStore() *Store {
	configService := configServiceImpl{}
	storage := configService.GetStorage()
	editor := configService.GetEditor()

	store := &Store{
		storage:          storage,
		editor:           editor,
		notesDictionary:  make(map[string]Note),
		configService:    configService,
		clipboardService: clipboardServiceImpl{},
	}

	return store
}

func (s Store) GetNotes() []Note {
	return s.notes
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

	uniqueName := s.generateUniqueName(note.Name)
	err := s.saveNote(uniqueName, note)

	if err == nil {
		s.currentNoteName = uniqueName
	}

	return err
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

func (s *Store) UpdateCurrentNoteContent(newContent string) error {
	if note, ok := s.GetCurrentNote(); ok {
		note.Content = newContent
		note.UpdatedAt = time.Now()

		err := s.saveNote(note.Name, note)

		if err != nil {
			return err
		}

		s.notesDictionary[note.Name] = note

		s.notes = slices.DeleteFunc(s.notes, func(n Note) bool {
			return n.Name == note.Name
		})

		s.notes = append([]Note{note}, s.notes...)

		return nil
	}

	return errors.New("note not found")
}

func (s *Store) RenameCurrentNote(newName string) (Note, error) {
	if note, ok := s.GetCurrentNote(); ok {
		if renamedNote, err := s.RenameNote(note.Name, newName); err == nil {
			s.SetCurrentNoteName(renamedNote.Name)
			return renamedNote, nil
		}
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

func (s Store) GetEditor() string {
	return s.configService.GetEditor()
}

func (s *Store) SetEditor(editor string) error {
	err := s.configService.SetEditor(editor)

	if err != nil {
		return err
	}

	s.editor = editor
	return nil
}

func (s Store) GetNotePath(name string) string {
	return filepath.Join(s.storage, name+".md")
}

// saveNote saves a note to the store
func (s *Store) saveNote(name string, note Note) error {
	path := filepath.Join(s.storage, name+".md")

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
	originalName := name
	counter := 1

	for {
		if _, exists := s.notesDictionary[strings.ToLower(name)]; !exists {
			break
		}

		name = originalName + "-" + strconv.Itoa(counter)
		counter++
	}

	return name
}

// func getCreationTime(filePath string) (time.Time, error) {
// 	info, err := os.Stat(filePath)
// 	if err != nil {
// 		return time.Time{}, err
// 	}

// 	stat, ok := info.Sys().(*syscall.Stat_t)
// 	if !ok {
// 		return time.Time{}, fmt.Errorf("failed to get raw syscall.Stat_t")
// 	}

// 	// Different OS has different fields in syscall.Stat_t
// 	switch runtime.GOOS {
// 	case "darwin":
// 		// macOS has Birthtimespec for creation time
// 		return time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec)), nil
// 	case "windows":
// 		// Windows implementation would be different and should use syscall.GetFileTime
// 		return time.Time{}, fmt.Errorf("use separate Windows implementation")
// 	default:
// 		// Linux generally doesn't store true creation time, using Ctim (status change time)
// 		return time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec)), nil
// 	}
// }

func (s *Store) loadNoteFromFile(path string) (Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Note{}, fmt.Errorf("failed to read note file: %w", err)
	}

	content := strings.TrimSuffix(string(data), "\n")

	name := strings.TrimSuffix(filepath.Base(path), ".md")

	fileInfo, err := os.Stat(path)

	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return Note{}, err
	}

	// Creation time is not supported on all platforms
	// TODO: Investigate if we can get the creation time on all platforms
	// createdAt, err := getCreationTime(path)

	// if err != nil {
	// 	fmt.Printf("Error getting file creation time: %v\n", err)
	// 	return Note{}, err
	// }

	updatedAt := fileInfo.ModTime()

	return Note{
		Name:      name,
		Content:   content,
		UpdatedAt: updatedAt,
		Byte:      data,
	}, nil
}
