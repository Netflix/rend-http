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
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"

	"github.com/netflix/rend-http/httph"
	"github.com/netflix/rend/handlers"
	"github.com/netflix/rend/metrics"
	"github.com/netflix/rend/orcas"
	"github.com/netflix/rend/protocol"
	"github.com/netflix/rend/protocol/binprot"
	"github.com/netflix/rend/protocol/textprot"
	"github.com/netflix/rend/server"
)

func init() {
	// Setting up signal handlers
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		<-sigs
		panic("Keyboard Interrupt")
	}()

	// http debug and metrics endpoint
	go http.ListenAndServe("localhost:11299", nil)

	// metrics output prefix
	metrics.SetPrefix("rend_http_")
}

type proxyinfo struct {
	listenPort int
	proxyHost  string
	proxyPort  int
	cacheName  string
}

var pis = []proxyinfo{}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Proxies a list of memcached protocol ports to a corresponding list of proxy hostnames, ports, and caches.\n")
		flag.PrintDefaults()
	}

	var listenPortsStr string
	var proxyHostsStr string
	var proxyPortsStr string
	var cacheNamesStr string

	flag.StringVar(&listenPortsStr, "listen-ports", "", "List of TCP ports to proxy from, separated by '|'")
	flag.StringVar(&proxyHostsStr, "proxy-hosts", "", "List of hostnames to proxy to, separated by '|'")
	flag.StringVar(&proxyPortsStr, "proxy-ports", "", "List of ports to proxy to, separated by '|'")
	flag.StringVar(&cacheNamesStr, "cache-names", "", "List of cache names to proxy to, separated by '|'")

	flag.Parse()

	if len(listenPortsStr) == 0 || len(proxyHostsStr) == 0 || len(proxyPortsStr) == 0 || len(cacheNamesStr) == 0 {
		log.Fatalln("Error: Must provide all params: --listen-ports, --proxy-hosts, --proxy-ports, --cache-names.")
	}

	// Trim any quotes off of the args
	trimQuotes := func(r rune) bool { return r == '"' }

	listenPortsStr = strings.TrimFunc(listenPortsStr, trimQuotes)
	proxyHostsStr = strings.TrimFunc(proxyHostsStr, trimQuotes)
	proxyPortsStr = strings.TrimFunc(proxyPortsStr, trimQuotes)
	cacheNamesStr = strings.TrimFunc(cacheNamesStr, trimQuotes)

	listenPortsParts := strings.Split(listenPortsStr, "|")
	listenPorts := make([]int, len(listenPortsParts))
	for i, p := range listenPortsParts {
		trimmed := strings.TrimSpace(p)
		if len(trimmed) == 0 {
			log.Fatalln("Error: Invalid ports; must not have blank entries.")
		}
		temp, err := strconv.Atoi(trimmed)
		if err != nil {
			log.Fatalf("Error: Invalid port: %s", trimmed)
		}
		listenPorts[i] = temp
	}

	proxyHosts := strings.Split(proxyHostsStr, "|")
	for i, u := range proxyHosts {
		proxyHosts[i] = strings.TrimSpace(u)
		if len(proxyHosts[i]) == 0 {
			log.Fatalln("Error:Invalid domain sockets; must not have blank entries.")
		}
	}

	proxyPortsParts := strings.Split(proxyPortsStr, "|")
	proxyPorts := make([]int, len(proxyPortsParts))
	for i, p := range proxyPortsParts {
		trimmed := strings.TrimSpace(p)
		if len(trimmed) == 0 {
			log.Fatalln("Error: Invalid proxy ports; must not have blank entries.")
		}
		temp, err := strconv.Atoi(trimmed)
		if err != nil {
			log.Fatalf("Error: Invalid port: %s", trimmed)
		}
		proxyPorts[i] = temp
	}

	cacheNames := strings.Split(cacheNamesStr, "|")
	for i, u := range cacheNames {
		cacheNames[i] = strings.TrimSpace(u)
		if len(cacheNames[i]) == 0 {
			log.Fatalln("Error:Invalid domain sockets; must not have blank entries.")
		}
	}

	if len(listenPorts) != len(proxyHosts) || len(listenPorts) != len(proxyPorts) || len(listenPorts) != len(cacheNames) {
		log.Fatalf("Error: all lists must match in length. Got %d listen ports, %d proxy hosts, %d proxy ports, %d cache names\n",
			len(listenPorts), len(proxyHosts), len(proxyPorts), len(cacheNames))
	}

	for i := 0; i < len(listenPorts); i++ {
		pis = append(pis, proxyinfo{
			listenPort: listenPorts[i],
			proxyHost:  proxyHosts[i],
			proxyPort:  proxyPorts[i],
			cacheName:  cacheNames[i],
		})
	}
}

func main() {
	for _, pi := range pis {
		largs := server.ListenArgs{
			Type: server.ListenTCP,
			Port: pi.listenPort,
		}

		go server.ListenAndServe(
			largs,
			[]protocol.Components{binprot.Components, textprot.Components},
			server.Default,
			orcas.L1Only,
			httph.New(pi.proxyHost, pi.proxyPort, pi.cacheName),
			handlers.NilHandler,
		)
	}

	runtime.Goexit()
}
