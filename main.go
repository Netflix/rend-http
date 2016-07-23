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

package main

import (
	"flag"

	"github.com/netflix/rend-http/httph"
	"github.com/netflix/rend/handlers"
	"github.com/netflix/rend/orcas"
	"github.com/netflix/rend/server"
)

var (
	proxyHost string
	proxyPort int
	cacheName string
)

func init() {
	flag.StringVar(&proxyHost, "proxy-host", "localhost", "Host to proxy traffic to")
	flag.IntVar(&proxyPort, "proxy-host", 9001, "Port on host to proxy traffic to")
	flag.StringVar(&cacheName, "cache-name", "evcache", "The cache name to proxy traffic to")
}

func main() {
	flag.Parse()

	largs := server.ListenArgs{
		Type: server.ListenTCP,
		Port: 11211,
	}

	server.ListenAndServe(
		largs,
		server.Default,
		orcas.L1Only,
		httph.New(proxyHost, proxyPort, cacheName),
		handlers.NilHandler,
	)
}
