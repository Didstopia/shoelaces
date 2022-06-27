// Copyright 2018 ThousandEyes Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"os"
	"path"
	"sync"
	"time"

	"github.com/Didstopia/shoelaces/internal/log"

	"github.com/fsnotify/fsnotify"
)

const (
	// InitTarget is an initial dummy target assigned to the servers
	InitTarget = "NOTARGET"
)

// Server holds data that uniquely identifies a server
type Server struct {
	Mac      string
	IP       string
	Hostname string
}

// Servers is an array of Server
type Servers []Server

// Len implementation for the sort Interface
func (s Servers) Len() int {
	return len(s)
}

// Swap implementation for the sort interface
func (s Servers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less implementation for the Sort interface
func (s Servers) Less(i, j int) bool {
	return s[i].Mac < s[j].Mac
}

// State holds information regarding a host that is attempting to boot.
type State struct {
	Server
	Target      string
	Environment string
	Params      map[string]interface{}
	Retry       int
	LastAccess  int
}

// States holds a map between MAC addresses and
// States. It provides a mutex for thread-safety.
type States struct {
	sync.RWMutex
	Servers map[string]*State
}

// New returns a Server with is values initialized
func New(mac string, ip string, hostname string) Server {
	return Server{
		Mac:      mac,
		IP:       ip,
		Hostname: hostname,
	}
}

// AddServer adds a server to the States struct
func (m *States) AddServer(server Server) {
	m.Servers[server.Mac] = &State{
		Server:     server,
		Target:     InitTarget,
		Retry:      1,
		LastAccess: int(time.Now().UTC().Unix()),
	}
}

// DeleteServer deletes a server from the States struct
func (m *States) DeleteServer(mac string) {
	delete(m.Servers, mac)
}

// StartStateCleaner spawns a goroutine that cleans MAC addresses that
// have been inactive in Shoelaces for more than 3 minutes.
func StartStateCleaner(logger log.Logger, serverStates *States) {
	const (
		// 3 minutes
		expireAfterSec = 3 * 60
		cleanInterval  = time.Minute
	)
	// Clean up the server states. Expire after 3 minutes
	go func() {
		for {
			time.Sleep(cleanInterval)

			servers := serverStates.Servers
			expire := int(time.Now().UTC().Unix()) - expireAfterSec

			logger.Debug("component", "polling", "msg", "Cleaning", "before", time.Unix(int64(expire), 0))

			serverStates.Lock()
			for mac, state := range servers {
				if state.LastAccess <= expire {
					delete(servers, mac)
					logger.Debug("component", "polling", "msg", "Mac cleaned", "mac", mac)
				}
			}
			serverStates.Unlock()
		}
	}()
}

// TODO: Should this be in its own file instead?
func WatchStuff(logger log.Logger, dataDir string, mappingsFile string, initMappings func(string) error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Failed to create watcher: ", err)
		os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Log the file change event
				logger.Debug("component", "watcher", "msg", "File changed", "file", event.Name, "type", event.Op)

				// Check if the change was a write event
				if event.Op&fsnotify.Write == fsnotify.Write {
					// Check if the file is the mappings file
					mappingsPath := path.Join(dataDir, mappingsFile)
					if event.Name == mappingsPath {
						// Mappings file changed, so we will attempt to reload all mappings
						logger.Info("component", "watcher", "msg", "Mappings file changed, recreating mappings")
						if err := initMappings(mappingsPath); err != nil {
							logger.Error("component", "watcher", "msg", "Init mappings error:", err)
							os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
						}
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("component", "watcher", "msg", "Watcher error:", err)
				os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
			}
		}
	}()

	// FIXME: fsnotify/watcher is NOT recursive, so we need to add every directory and subdirectory we want to watch,
	//        which includes any newly created directories, but also removing any removed ones!

	// TODO: Do we even need to watch the data directory? Wouldn't watching mappings.yaml be enough?
	// Watch the data directory
	// if err := watcher.Add(env.DataDir); err != nil {
	// 	env.Logger.Error("component", "watcher", "msg", "Failed to watch data directory:", err)
	// 	os.Exit(1)
	// }

	// Register the mappings file in the filesystem watcher
	if err := watcher.Add(path.Join(dataDir, mappingsFile)); err != nil {
		logger.Error("component", "watcher", "msg", "Failed to watch mappings file:", err)
		os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
	}

	// FIXME: We need a way to gracefully shut this down, passing in a context or channel for example?
	logger.Info("component", "watcher", "msg", "Watching for changes...")
	<-done
}
