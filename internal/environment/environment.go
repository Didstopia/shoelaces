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

package environment

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Didstopia/shoelaces/internal/event"
	"github.com/Didstopia/shoelaces/internal/log"
	"github.com/Didstopia/shoelaces/internal/mappings"
	"github.com/Didstopia/shoelaces/internal/server"
	"github.com/Didstopia/shoelaces/internal/templates"
	"github.com/fsnotify/fsnotify"
)

// Environment struct holds the shoelaces instance global data.
type Environment struct {
	ConfigFile      string
	HostnameMaps    []mappings.HostnameMap
	NetworkMaps     []mappings.NetworkMap
	ServerStates    *server.States
	EventLog        *event.Log
	ParamsBlacklist []string
	Templates       *templates.ShoelacesTemplates // Dynamic slc templates
	StaticTemplates *template.Template            // Static Templates
	Environments    []string                      // Valid config environments
	Logger          log.Logger

	BindAddr          string
	BaseURL           string
	DataDir           string
	StaticDir         string
	EnvDir            string
	TemplateExtension string
	MappingsFile      string
	Debug             bool
}

// New returns an initialized environment structure
func New() *Environment {
	env := defaultEnvironment()
	env.setFlags()
	env.validateFlags()

	if env.Debug {
		env.Logger = log.AllowDebug(env.Logger)
	}

	if env.BaseURL == "" {
		env.BaseURL = env.BindAddr
	}

	env.Environments = env.initEnvOverrides()

	env.EventLog = &event.Log{}

	env.Logger.Info("component", "environment", "msg", "Override found", "environment", env.Environments)

	mappingsPath := path.Join(env.DataDir, env.MappingsFile)
	if err := env.initMappings(mappingsPath); err != nil {
		panic(err)
	}

	env.initStaticTemplates()
	env.Templates.ParseTemplates(env.Logger, env.DataDir, env.EnvDir, env.Environments, env.TemplateExtension)
	server.StartStateCleaner(env.Logger, env.ServerStates)

	// FIXME: Pass in a context so we can cancel the goroutine and gracefully shut it down!
	// go server.WatchStuff(env, env.Logger, env.DataDir, env.MappingsFile, env.initMappings)
	go watchStuff(env)

	return env
}

func defaultEnvironment() *Environment {
	env := &Environment{}
	env.NetworkMaps = make([]mappings.NetworkMap, 0)
	env.HostnameMaps = make([]mappings.HostnameMap, 0)
	// FIXME: This whatever the issue is with this warning?!
	env.ServerStates = &server.States{sync.RWMutex{}, make(map[string]*server.State)}
	env.ParamsBlacklist = []string{"baseURL"}
	env.Templates = templates.New()
	env.Environments = make([]string, 0)
	env.Logger = log.MakeLogger(os.Stdout)

	return env
}

func (env *Environment) initStaticTemplates() {
	staticTemplates := []string{
		path.Join(env.StaticDir, "templates/html/header.html"),
		path.Join(env.StaticDir, "templates/html/index.html"),
		path.Join(env.StaticDir, "templates/html/events.html"),
		path.Join(env.StaticDir, "templates/html/mappings.html"),
		path.Join(env.StaticDir, "templates/html/footer.html"),
	}

	fmt.Println(env.StaticDir)

	for _, t := range staticTemplates {
		if _, err := os.Stat(t); err != nil {
			env.Logger.Error("component", "environment", "msg", "Template does not exists!", "environment", t)
			os.Exit(1)
		}
	}

	env.StaticTemplates = template.Must(template.ParseFiles(staticTemplates...))
}

func (env *Environment) initEnvOverrides() []string {
	var environments = make([]string, 0)
	envPath := filepath.Join(env.DataDir, env.EnvDir)
	files, err := ioutil.ReadDir(envPath)
	if err == nil {
		for _, f := range files {
			if f.IsDir() {
				environments = append(environments, f.Name())
			}
		}
	}
	return environments
}

func (env *Environment) initMappings(mappingsPath string) error {
	configMappings := mappings.ParseYamlMappings(env.Logger, mappingsPath)

	// Ensure env.NetworkMaps is empty
	env.NetworkMaps = make([]mappings.NetworkMap, 0)

	for _, configNetMap := range configMappings.NetworkMaps {
		_, ipnet, err := net.ParseCIDR(configNetMap.Network)
		if err != nil {
			return err
		}

		netMap := mappings.NetworkMap{Network: ipnet, Script: initScript(configNetMap.Script)}
		env.NetworkMaps = append(env.NetworkMaps, netMap)
	}

	// Ensure env.HostnameMaps is empty
	env.HostnameMaps = make([]mappings.HostnameMap, 0)

	for _, configHostMap := range configMappings.HostnameMaps {
		regex, err := regexp.Compile(configHostMap.Hostname)
		if err != nil {
			return err
		}

		hostMap := mappings.HostnameMap{Hostname: regex, Script: initScript(configHostMap.Script)}
		env.HostnameMaps = append(env.HostnameMaps, hostMap)
	}

	return nil
}

func initScript(configScript mappings.YamlScript) *mappings.Script {
	mappingScript := &mappings.Script{
		Name:        configScript.Name,
		Environment: configScript.Environment,
		Params:      make(map[string]interface{}),
	}
	for key := range configScript.Params {
		mappingScript.Params[key] = configScript.Params[key]
	}

	return mappingScript
}

var validDataDirs = []string{
	"cloud-config",
	"env_overrides",
	"ipxe",
	"kickstart",
	"preseed",
	"static",
}

func isValidDataDir(dir string) bool {
	for _, validDir := range validDataDirs {
		if strings.Contains(dir, "/"+validDir+"/") {
			return true
		}
	}
	return false
}

// FIXME: This only seems to work for mappings.yaml, but NOT for eg. foo.slc etc.
//
//	although I THINK this is because it's simply not recursive?
//
// TODO: Should this be in its own file instead?
// func watchStuff(logger log.Logger, dataDir string, mappingsFile string, initMappings func(string) error) {
func watchStuff(env *Environment) {
	logger := env.Logger
	dataDir := env.DataDir
	mappingsFile := env.MappingsFile
	// initMappings := env.InitMappings

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
						if err := env.initMappings(mappingsPath); err != nil {
							logger.Error("component", "watcher", "msg", "Init mappings error:", err)
							os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
						}
						// Otherwise we check for the data directories and rebuild the templates if they change
					} else if isValidDataDir(event.Name) {
						// TODO: We probably need to reload more than just the ".slc" templates, eg. re-initializing env_overrides etc. ?
						logger.Info("component", "watcher", "msg", "Data directory changed, rebuilding templates")
						// TODO: Reload templates
						env.Templates.ParseTemplates(env.Logger, env.DataDir, env.EnvDir, env.Environments, env.TemplateExtension)
					} else {
						logger.Info("component", "watcher", "msg", "Unknown change detected:", event.Name)
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

	// Register the cloud-config directory in the filesystem watcher
	if err := watcher.Add(path.Join(dataDir, "cloud-config")); err != nil {
		logger.Error("component", "watcher", "msg", "Failed to watch cloud-config directory:", err)
		os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
	}

	// Register the env_overrides directory in the filesystem watcher
	if err := watcher.Add(path.Join(dataDir, "env_overrides")); err != nil {
		logger.Error("component", "watcher", "msg", "Failed to watch env_overrides directory:", err)
		os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
	}

	// Register the ipxe directory in the filesystem watcher
	if err := watcher.Add(path.Join(dataDir, "ipxe")); err != nil {
		logger.Error("component", "watcher", "msg", "Failed to watch ipxe directory:", err)
		os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
	}

	// Register the preseed directory in the filesystem watcher
	if err := watcher.Add(path.Join(dataDir, "preseed")); err != nil {
		logger.Error("component", "watcher", "msg", "Failed to watch preseed directory:", err)
		os.Exit(1) // TODO: This probably doesn't allow us to do graceful shutdown?
	}

	// TODO: No need to watch for the static directory, right?

	// FIXME: We need a way to gracefully shut this down, passing in a context or channel for example?
	logger.Info("component", "watcher", "msg", "Watching for changes...")
	<-done
}
