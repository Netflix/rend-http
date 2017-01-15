// Copyright 2016 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	endpointPath = "/config"
)

// In this (small) universe, all config is an int
type conf map[string]int

func copyConf(orig conf) conf {
	ret := make(conf)
	for k, v := range orig {
		ret[k] = v
	}
	return ret
}

var confHolder atomic.Value

func init() {
	confHolder.Store(make(conf))

	http.Handle(endpointPath, http.HandlerFunc(handleConfig))
	http.Handle(endpointPath+"/", http.HandlerFunc(handleConfig))
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	// Allow GET to retrieve a specific config (or all)
	// Allow PUT to set a specific config
	// Otherwise return a 405 Method Not Allowed

	switch r.Method {
	case "GET":
		// read from value then map
		// If the path does not include a key, print all config
		// If it does include a key, use it to look up in the map and print that value

		// Take the "/config" off the front
		key := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, endpointPath))
		c := confHolder.Load().(conf)

		if key == "" || key == "/" {
			// print all config
			for k, v := range c {
				fmt.Fprintf(w, "%s %d\n", k, v)
			}

		} else {
			// print specific config
			// assume the rest of the path is the config key

			// skip the leading slash
			key = key[1:]

			if v, ok := c[key]; ok {
				fmt.Fprintf(w, "%s %d\n", key, v)
			} else {
				w.WriteHeader(404)
			}
		}

	case "PUT":
		// read from value, copy config, modify copy, store copy
		// The body of the request is assumed to be a string that can be converted into an int
		// Otherwise the handler will return 400

		// Translate path into a key
		key := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, endpointPath))
		if key == "" || key == "/" {
			w.WriteHeader(400)
			return
		}

		// skip the / at the beginning
		key = key[1:]

		// Extract body and turn into int
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		r.Body.Close()

		v, err := strconv.Atoi(string(raw))
		if err != nil {
			w.WriteHeader(500)
			return
		}

		c := confHolder.Load().(conf)
		c = copyConf(c)
		c[key] = v

		confHolder.Store(c)

	default:
		w.WriteHeader(405)
	}
}

// Get retrieves the value from the configuration or, if it's not explicitly set, returns otherwise
func Get(key string, otherwise int) int {
	c := confHolder.Load().(conf)

	if v, ok := c[key]; ok {
		return v
	}

	return otherwise
}
