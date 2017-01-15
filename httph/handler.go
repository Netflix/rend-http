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

package httph

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"time"

	"github.com/netflix/rend-http/config"
	"github.com/netflix/rend/common"
	"github.com/netflix/rend/handlers"
)

const (
	// NumTriesConfigName is the name of the dynamic config for the number of tries
	NumTriesConfigName = "numTries"

	// DefaultNumTries is the number of times the handler will try operations before
	// returning an error to the client
	DefaultNumTries = 4
	// 4 tries == 3 retries

	// RetryDelayMultiplierConfigName is the name of the dynamic config for the retry delay multiplier
	RetryDelayMultiplierConfigName = "retryDelayMultiplier"

	// DefaultRetryDelayMultiplier is the default multiplier for the retry wait time,
	// It is set up to wait for 10, 40, 90, etc ms successively on retries
	DefaultRetryDelayMultiplier = 10
)

func retryDelay(try int) {
	// wait for 10, 40, and 90 ms successively on retries
	if try > 0 {
		mult := config.Get(RetryDelayMultiplierConfigName, DefaultRetryDelayMultiplier)
		<-time.After(time.Duration(try) * time.Millisecond * time.Duration(mult))
	}
}

// Handler implements the github.com/netflix/rend/handlers.Handler interface.
// The only operations supported right now are set, get, and delete.
type Handler struct {
	urlprefix string
	client    http.Client
}

// New creates a new handler constructor function. The returned function returns
// the same exact singleton instance of the Handler every time. This means that
// all requests will be able to take advantage of the http keepalive on the conn
// pool to the http proxy.
func New(host string, port int, cache string) handlers.HandlerConst {
	singleton := &Handler{
		urlprefix: fmt.Sprintf("http://%s:%d/evcrest/v1.0/%s/", host, port, cache),
		client:    http.Client{},
	}

	return func() (handlers.Handler, error) {
		return singleton, nil
	}
}

func (h *Handler) makeURL(key []byte) string {
	return h.urlprefix + string(key)
}

// Set performs an HTTP PUT request on the backend server
func (h *Handler) Set(cmd common.SetRequest) error {
	url := h.makeURL(cmd.Key) + "?ttl=" + strconv.Itoa(int(cmd.Exptime))

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	tries := config.Get(NumTriesConfigName, DefaultNumTries)
	for i := 0; i < tries; i++ {
		retryDelay(i)

		// Reset body
		req.Body = ioutil.NopCloser(bytes.NewReader(cmd.Data))

		res, err := h.client.Do(req)
		if err != nil {
			return err
		}

		// discard and close body to allow reuse of connection
		if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
			return err
		}
		res.Body.Close()

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return nil
		}

		// Shortcut on errors that are going to fail on subsequent tries
		if res.StatusCode == 400 || res.StatusCode == 500 {
			return common.ErrInternal
		}

		log.Printf("[SET] Unexpected status code in HTTP response: %d\n", res.StatusCode)
		log.Printf("[SET] url: %s\n", url)
	}

	return common.ErrInternal
}

// Delete performs an HTTP DELETE request on the backend server
func (h *Handler) Delete(cmd common.DeleteRequest) error {
	url := h.makeURL(cmd.Key)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		// this would be a bad host, port, or cache
		return err
	}

	tries := config.Get(NumTriesConfigName, DefaultNumTries)
	for i := 0; i < tries; i++ {
		retryDelay(i)

		res, err := h.client.Do(req)
		if err != nil {
			return err
		}

		// discard and close body to allow reuse of connection
		if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
			return err
		}
		res.Body.Close()

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return nil
		}

		// Shortcut on failures where subsequent requests will fail
		if res.StatusCode == 500 {
			return common.ErrInternal
		}

		log.Printf("[DELETE] Unexpected status code in HTTP response: %d\n", res.StatusCode)
		log.Printf("[DELETE] url: %s\n", url)
	}

	return common.ErrInternal
}

// Get performs an HTTP GET request on the backend server for each key given
func (h *Handler) Get(cmd common.GetRequest) (<-chan common.GetResponse, <-chan error) {
	dataOut := make(chan common.GetResponse)
	errorOut := make(chan error)
	go realHandleGet(h, cmd, dataOut, errorOut)
	return dataOut, errorOut
}

func realHandleGet(h *Handler, cmd common.GetRequest, dataOut chan common.GetResponse, errorOut chan error) {
	defer close(errorOut)
	defer close(dataOut)

outer:
	for idx, key := range cmd.Keys {
		url := h.makeURL(key)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			errorOut <- err
			return
		}

		tries := config.Get(NumTriesConfigName, DefaultNumTries)
		for i := 0; i < tries; i++ {
			retryDelay(i)

			res, err := h.client.Do(req)
			if err != nil {
				errorOut <- err
				return
			}

			data, err := ioutil.ReadAll(res.Body)

			// Close body to allow reuse of connection
			res.Body.Close()

			switch res.StatusCode {
			case 200:
				dataOut <- common.GetResponse{
					Miss:   false,
					Quiet:  cmd.Quiet[idx],
					Opaque: cmd.Opaques[idx],
					Flags:  0,
					Key:    key,
					Data:   data,
				}

				continue outer

			case 404:
				dataOut <- common.GetResponse{
					Miss:   true,
					Quiet:  cmd.Quiet[idx],
					Opaque: cmd.Opaques[idx],
					Flags:  0,
					Key:    key,
				}

				continue outer

			case 500:
				// Don't retry for a request that will very likely fail
				errorOut <- common.ErrInternal
				return

			default:
				log.Printf("[GET] Unexpected status code in HTTP response: %d\n", res.StatusCode)
				log.Printf("[GET] url: %s\n", url)
			}
		}

		errorOut <- common.ErrInternal
		return
	}
}

// Close does nothing on this handler because they all share the same singleton
func (h *Handler) Close() error {
	// nothing to "close" here
	return nil
}

/////////////////////////////////////
// All the rest just return an error
/////////////////////////////////////

// Add is not implemented and returns common.ErrUnknownCmd
func (h *Handler) Add(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

// Replace is not implemented and returns common.ErrUnknownCmd
func (h *Handler) Replace(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

// Append is not implemented and returns common.ErrUnknownCmd
func (h *Handler) Append(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

// Prepend is not implemented and returns common.ErrUnknownCmd
func (h *Handler) Prepend(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

// GetE is not implemented and returns common.ErrUnknownCmd
func (h *Handler) GetE(cmd common.GetRequest) (<-chan common.GetEResponse, <-chan error) {
	errchan := make(chan error, 1)
	errchan <- common.ErrUnknownCmd
	return nil, errchan
}

// GAT is not implemented and returns common.ErrUnknownCmd
func (h *Handler) GAT(cmd common.GATRequest) (common.GetResponse, error) {
	return common.GetResponse{}, common.ErrUnknownCmd
}

// Touch is not implemented and returns common.ErrUnknownCmd
func (h *Handler) Touch(cmd common.TouchRequest) error {
	return common.ErrUnknownCmd
}
