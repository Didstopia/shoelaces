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

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Didstopia/shoelaces/internal/environment"
	"github.com/Didstopia/shoelaces/internal/handlers"
	"github.com/Didstopia/shoelaces/internal/router"
	// cp "github.com/otiai10/copy"
)

// These will be filled in at compile time
var version = "LATEST"
var build = "development build"

func main() {
	// Print the version and build information on startup
	fmt.Println("Shoelaces " + version + " (" + build + ")")

	// Prepare the environment
	env := environment.New()
	// prepareEnvironment(env)

	// Create the application, including the web server, request routers and handlers etc.
	app := handlers.MiddlewareChain(env).Then(router.ShoelacesRouter(env))

	// TODO: Here we would need to control our own context and pass it to everything,
	//       so that we have full control over the shutdown process and can do it gracefully!
	// Start the web server and wait for it to exit
	env.Logger.Info("component", "main", "transport", "http", "addr", env.BindAddr, "msg", "Listening for incoming HTTP requests")
	env.Logger.Error("component", "main", "err", http.ListenAndServe(env.BindAddr, app))

	os.Exit(1)
}

// FIXME: Abandoned this for now, as this should run BEFORE environment.New(),
//        as the environment needs to be setup BEFORE this, but this also means
//        that we don't have immediate access to user-configured parameters, like paths etc.

// Prepares the environment for running Shoelaces
// (replaces `docker_entrypoint.sh`)
// func prepareEnvironment(env *environment.Environment) {
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		panic(err)
// 	}

// 	// TODO: What if we embed the default mappings.yaml with the binary, then
// 	//       we can simply write it to the data directory if it's missing?!
// 	// TODO: If env.DataDir is empty or doesn't exist, copy /shoelaces_default/mappings.yaml to it
// 	// Check if env.DataDir exists or is empty
// 	log.Println("Checking if data directory exists or is empty")
// 	if _, err := os.Stat(env.DataDir); os.IsNotExist(err) {
// 		// DataDir doesn't exist, create it
// 		log.Println("Data directory doesn't exist, creating it")
// 		err := os.MkdirAll(env.DataDir, 0755)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		// Check if mappings.yaml exists in the env.DataDir directory
// 		log.Println("Checking if mappings.yaml exists in the data directory")
// 		if _, err := os.Stat(path.Join(env.DataDir, "mappings.yaml")); os.IsNotExist(err) {
// 			// mappings.yaml doesn't exist, copy it from the default directory
// 			log.Println("Copying default mappings.yaml to", env.DataDir)
// 			err := cp.Copy(path.Join(cwd, "configs", "data-dir", "mappings.yaml"), path.Join(env.DataDir, "mappings.yaml"))
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			log.Println("Copied mappings.yaml from", path.Join(cwd, "configs", "data-dir", "mappings.yaml"), "to", path.Join(env.DataDir, "mappings.yaml"))
// 		}
// 	}

// 	// TODO: Could we also embed the _entire_ static website with the binary,
// 	//			 then write it to the static/web directory, just like above?
// 	// TODO: If env.StaticDir is empty or doesn't exist, copy /shoelaces/default/web/* to it
// 	// Check if env.StaticDir exists or is empty
// 	log.Println("Checking if env.StaticDir exists or is empty")
// 	if _, err := os.Stat(env.StaticDir); os.IsNotExist(err) {
// 		// Create the directory
// 		log.Println("Creating directory:", env.StaticDir)
// 		if err := os.Mkdir(env.StaticDir, 0755); err != nil {
// 			panic(err)
// 		}
// 		log.Println("Created directory", env.StaticDir)
// 		log.Println("Copying default static contents to", env.StaticDir)
// 		// FIXME: The "web/" likely won't work here..
// 		if err := cp.Copy(path.Join(cwd, "web/"), env.StaticDir); err != nil {
// 			panic(err)
// 		}
// 		log.Println("Copied", path.Join(cwd, "web/"), "to", env.StaticDir)
// 	}
// }
