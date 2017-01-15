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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

import "os"

func TestMain(m *testing.M) {
	go http.ListenAndServe("127.0.0.1:55555", nil)
	os.Exit(m.Run())
}

func TestHTTPEndpoint(t *testing.T) {
	t.Run("GET", func(t *testing.T) {
		t.Run("All", func(t *testing.T) {
			confHolder.Store(conf{
				"foo": 4,
				"bar": 5,
			})

			res, err := http.Get("http://localhost:55555/config")
			if err != nil {
				t.Fatalf("Got error during http get: %v", err)
			}

			if res.StatusCode != 200 {
				t.Fatalf("Expected status code of 200 but got %v", res.StatusCode)
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("Got error reading get response: %v", err)
			}

			if string(body) != "foo 4\nbar 5\n" {
				t.Fatalf("Got bad response: %v", string(body))
			}
		})
		t.Run("Single", func(t *testing.T) {
			confHolder.Store(conf{
				"foo": 4,
				"bar": 5,
			})

			res, err := http.Get("http://localhost:55555/config/foo")
			if err != nil {
				t.Fatalf("Got error during http get: %v", err)
			}

			if res.StatusCode != 200 {
				t.Fatalf("Expected status code of 200 but got %v", res.StatusCode)
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("Got error reading get response: %v", err)
			}

			if string(body) != "foo 4\n" {
				t.Fatalf("Got bad response: %v", string(body))
			}
		})
		t.Run("Miss", func(t *testing.T) {
			confHolder.Store(conf{
				"foo": 4,
				"bar": 5,
			})

			res, err := http.Get("http://localhost:55555/config/baz")
			if err != nil {
				t.Fatalf("Got error during http get: %v", err)
			}

			if res.StatusCode != 404 {
				t.Fatalf("Expected status code of 404 but got %v", res.StatusCode)
			}
		})
	})
	t.Run("PUT", func(t *testing.T) {
		req, err := http.NewRequest("PUT", "http://localhost:55555/config/foo", strings.NewReader("4"))
		if err != nil {
			t.Fatalf("Got error during request creation (this is a bug in the test): %v", err)
		}

		res, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Got error performing request: %v", err)
		}

		if res.StatusCode != 200 {
			t.Fatalf("Expected status code of 200 but got %v", res.StatusCode)
		}

		c := confHolder.Load().(conf)

		if v, ok := c["foo"]; ok {
			if v != 4 {
				t.Fatalf("Expected stored value of 4 but got %v", v)
			}
		} else {
			t.Fatalf("Value was not stored in config. Whole config: %v", c)
		}
	})
	t.Run("RoundTrip", func(t *testing.T) {
		// Set
		req, err := http.NewRequest("PUT", "http://localhost:55555/config/foo", strings.NewReader("4"))
		if err != nil {
			t.Fatalf("Got error during request creation (this is a bug in the test): %v", err)
		}

		res, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Got error performing request: %v", err)
		}

		if res.StatusCode != 200 {
			t.Fatalf("Expected status code of 200 but got %v", res.StatusCode)
		}

		// Get
		res, err = http.Get("http://localhost:55555/config/foo")
		if err != nil {
			t.Fatalf("Got error during http get: %v", err)
		}

		if res.StatusCode != 200 {
			t.Fatalf("Expected status code of 200 but got %v", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Got error reading get response: %v", err)
		}

		if string(body) != "foo 4\n" {
			t.Fatalf("Got bad response: %v", string(body))
		}
	})
	t.Run("OtherMethods", func(t *testing.T) {
		req, err := http.NewRequest("PUT", "http://localhost:55555/config/foo", strings.NewReader("4"))
		if err != nil {
			t.Fatalf("Got error during request creation (this is a bug in the test): %v", err)
		}

		res, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Got error performing request: %v", err)
		}

		if res.StatusCode != 200 {
			t.Fatalf("Expected status code of 200 but got %v", res.StatusCode)
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("ExplicitlySet", func(t *testing.T) {
		confHolder.Store(conf{
			"foo": 4,
			"bar": 5,
		})

		v := Get("foo", 17)

		if v != 4 {
			t.Fatalf("Expected to get value 4 but got %v", v)
		}
	})
	t.Run("UsingOtherwise", func(t *testing.T) {
		confHolder.Store(conf{})

		v := Get("foo", 17)

		if v != 17 {
			t.Fatalf("Expected to get value 17 but got %v", v)
		}
	})
}
