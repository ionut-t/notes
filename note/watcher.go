package note

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watcherDebounce = 200 * time.Millisecond

// Watcher notifies when notes in the storage directory are created,
// changed, renamed or deleted, including changes made outside the app.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	changes   chan struct{}
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
		done:    make(chan struct{}),
	}

	if err := w.addRecursive(s.storage); err != nil {
		fsWatcher.Close()
		return nil, err
	}

	go w.run()

	return w, nil
}

// Changes emits once per burst of file system events.
func (w *Watcher) Changes() <-chan struct{} {
	return w.changes
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
					_ = w.addRecursive(event.Name)
					timer.Reset(watcherDebounce)
					continue
				}
			}

			if isRelevantEvent(event) {
				timer.Reset(watcherDebounce)
			}

		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}

		case <-timer.C:
			select {
			case w.changes <- struct{}{}:
			default:
			}
		}
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

// isRelevantEvent ignores editor swap/backup files and attribute-only changes.
// Removes and renames are always relevant since they may affect a directory.
func isRelevantEvent(event fsnotify.Event) bool {
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		return true
	}

	if event.Op == fsnotify.Chmod {
		return false
	}

	return strings.HasSuffix(event.Name, ".md")
}
