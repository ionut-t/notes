package note

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watcherDebounce = 200 * time.Millisecond

// Watcher notifies when notes in the storage directory are created,
// changed, renamed or deleted, including changes made outside the app.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	changes   chan struct{}
	errors    chan error
	done      chan struct{}
}

func (s *Store) Watch() (*Watcher, error) {
	if err := os.MkdirAll(s.storage, 0755); err != nil {
		return nil, err
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsWatcher,
		// buffered so that a pending notification is kept while the UI is busy
		changes: make(chan struct{}, 1),
		errors:  make(chan error, 1),
		done:    make(chan struct{}),
	}

	if err := w.addRecursive(s.storage); err != nil {
		return nil, errors.Join(err, fsWatcher.Close())
	}

	go w.run()

	return w, nil
}

// Changes emits once per burst of file system events.
func (w *Watcher) Changes() <-chan struct{} {
	return w.changes
}

// Errors emits failures that may cause changes to be missed, such as
// hitting the OS limit of watched directories.
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.fsWatcher.Close()
}

func (w *Watcher) run() {
	timer := time.NewTimer(watcherDebounce)
	timer.Stop()

	for {
		select {
		case <-w.done:
			timer.Stop()
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := w.addRecursive(event.Name); err != nil {
						w.reportError(fmt.Errorf("failed to watch %s: %w", event.Name, err))
					}
					timer.Reset(watcherDebounce)
					continue
				}
			}

			if isRelevantEvent(event) {
				timer.Reset(watcherDebounce)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}

			w.reportError(err)

		case <-timer.C:
			select {
			case w.changes <- struct{}{}:
			default:
			}
		}
	}
}

// reportError drops the error if one is already pending, so a burst of
// errors never blocks the event loop.
func (w *Watcher) reportError(err error) {
	select {
	case w.errors <- err:
	default:
	}
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return w.fsWatcher.Add(path)
		}

		return nil
	})
}

// isRelevantEvent ignores files that aren't notes, such as editor swap, backup
// and temporary files, and attribute-only changes. Removes and renames are
// always relevant since they may affect a directory. Saves that write a
// temporary file and rename it over the note are still caught by the Create
// event for the note itself.
func isRelevantEvent(event fsnotify.Event) bool {
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		return true
	}

	if event.Op == fsnotify.Chmod {
		return false
	}

	return isNoteFile(filepath.Base(event.Name))
}
