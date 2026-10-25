// Package devserver makes it easy to view statically generated websites and
// automatically rebuild them when source data changes. When combined with the
// "livejs" plugin, it is possible to have a live preview of your site.
package devserver

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Builder interface should be implemented by you to contain the required
// goldsmith chain to generate your website.
type Builder interface {
	Build(sourceDir, targetDir, cacheDir string)
}

// DevServe should be called to start a web server using the provided builder.
// While the source directory will be watched for changes by default, it is
// possible to pass in additional directories to watch; modification of these
// directories will automatically trigger a site rebuild. This function does
// not return and will continue watching for file changes and serving your
// website until it is terminated.
func DevServe(builder Builder, port int, sourceDir, targetDir, cacheDir string, watchDirs ...string) {
	dirs := append(append([]string(nil), watchDirs...), sourceDir)
	watcher, err := build(dirs, func() {
		builder.Build(sourceDir, targetDir, cacheDir)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	httpAddr := fmt.Sprintf(":%d", port)
	httpHandler := http.FileServer(http.Dir(targetDir))

	log.Fatal(http.ListenAndServe(httpAddr, httpHandler))
}

func build(dirs []string, callback func()) (*fsnotify.Watcher, error) {
	if callback == nil {
		return nil, errors.New("build callback cannot be nil")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if err := watch(dir, watcher); err != nil {
			watcher.Close()
			return nil, err
		}
	}

	go func() {
		defer watcher.Close()

		const debounce = 100 * time.Millisecond
		timer := time.NewTimer(debounce)
		defer timer.Stop()
		timerC := timer.C
		resetTimer := func() {
			if !timer.Stop() && timerC != nil {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(debounce)
			timerC = timer.C
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				resetTimer()

				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if os.IsNotExist(err) {
						continue
					}
					if err != nil {
						log.Printf("watch %s: %v", event.Name, err)
						continue
					}

					if info.IsDir() {
						if err := watch(event.Name, watcher); err != nil && !os.IsNotExist(err) {
							log.Printf("watch %s: %v", event.Name, err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("file watcher: %v", err)
			case <-timerC:
				timerC = nil
				callback()
			}
		}
	}()

	return watcher, nil
}

func watch(dir string, watcher *fsnotify.Watcher) error {
	if err := watcher.Add(dir); err != nil {
		return err
	}

	items, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, item := range items {
		fullPath := filepath.Join(dir, item.Name())
		if item.IsDir() {
			if err := watch(fullPath, watcher); err != nil {
				return err
			}
		}
	}

	return nil
}
