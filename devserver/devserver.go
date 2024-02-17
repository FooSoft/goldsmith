// Package devserver makes it easy to view statically generated websites and
// automatically rebuild them when source data changes. When combined with the
// "livejs" plugin, it is possible to have a live preview of your site.
package devserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
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
	dirs := append(watchDirs, sourceDir)
	build(dirs, func() { builder.Build(sourceDir, targetDir, cacheDir) })

	httpAddr := fmt.Sprintf(":%d", port)
	httpHandler := http.FileServer(http.Dir(targetDir))

	log.Fatal(http.ListenAndServe(httpAddr, httpHandler))
}

func build(dirs []string, callback func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	var mutex sync.Mutex
	timestamp := time.Now()
	dirty := true

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				mutex.Lock()
				timestamp = time.Now()
				dirty = true
				mutex.Unlock()

				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if os.IsNotExist(err) {
						continue
					}
					if err != nil {
						log.Fatal(err)
					}

					if info.IsDir() {
						watch(event.Name, watcher)
					} else {
						watcher.Add(event.Name)
					}
				}
			case err := <-watcher.Errors:
				log.Fatal(err)
			}
		}
	}()

	go func() {
		for range time.Tick(10 * time.Millisecond) {
			if dirty && time.Now().Sub(timestamp) > 100*time.Millisecond {
				mutex.Lock()
				dirty = false
				mutex.Unlock()

				callback()
			}
		}
	}()

	for _, dir := range dirs {
		watch(dir, watcher)
	}
}

func watch(dir string, watcher *fsnotify.Watcher) {
	watcher.Add(dir)

	items, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		fullPath := path.Join(dir, item.Name())
		if item.IsDir() {
			watch(fullPath, watcher)
		} else {
			watcher.Add(fullPath)
		}
	}
}
