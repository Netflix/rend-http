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

	"github.com/netflix/rend/common"
	"github.com/netflix/rend/handlers"
)

type Handler struct {
	urlprefix string
	client    http.Client
}

func New(host string, port int, cache string) handlers.HandlerConst {
	singleton := &Handler{
		urlprefix: fmt.Sprintf("http://%s:%d/evcrest/v1.0/%s/", host, port, cache),
		client:    http.Client{},
	}

	return func() (handlers.Handler, error) {
		return singleton, nil
	}
}

func (h *Handler) makeUrl(key []byte) string {
	return h.urlprefix + string(key)
}

func (h *Handler) Set(cmd common.SetRequest) error {
	url := h.makeUrl(cmd.Key) + "?ttl=" + strconv.Itoa(int(cmd.Exptime))
	res, err := h.client.Post(url, "application/octet-stream", bytes.NewBuffer(cmd.Data))
	if err != nil {
		return err
	}
	// discard body to allow reuse of connection
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		return err
	}

	switch res.StatusCode {
	case 202:
		return nil
	case 400:
		// this is a case that comes back from the server, but would only happen
		// if the code here is incorrect
		log.Println("Invalid request sent to POST endpoint as a part a set.")
		log.Printf("url: %s\n", url)
		return common.ErrInternal
	default:
		log.Printf("[SET] Unexpected status code in HTTP response: %d\n", res.StatusCode)
		log.Printf("[SET] url: %s\n", url)
		return common.ErrInternal
	}
}

func (h *Handler) Delete(cmd common.DeleteRequest) error {
	url := h.makeUrl(cmd.Key)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		// this would be a bad host, port, or cache
		return err
	}
	res, err := h.client.Do(req)
	if err != nil {
		return err
	}
	// discard body to allow reuse of connection
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		return err
	}

	switch res.StatusCode {
	case 200:
		return nil
	default:
		log.Printf("[DELETE] Unexpected status code in HTTP response: %d\n", res.StatusCode)
		log.Printf("[DELETE] url: %s\n", url)
		return common.ErrInternal
	}
}

func (h *Handler) Get(cmd common.GetRequest) (<-chan common.GetResponse, <-chan error) {
	dataOut := make(chan common.GetResponse)
	errorOut := make(chan error)
	go realHandleGet(h, cmd, dataOut, errorOut)
	return dataOut, errorOut
}

func realHandleGet(h *Handler, cmd common.GetRequest, dataOut chan common.GetResponse, errorOut chan error) {
	defer close(errorOut)
	defer close(dataOut)

	for idx, key := range cmd.Keys {
		url := h.makeUrl(key)
		res, err := h.client.Get(url)
		if err != nil {
			errorOut <- err
			return
		}

		data, err := ioutil.ReadAll(res.Body)

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
		case 404:
			dataOut <- common.GetResponse{
				Miss:   true,
				Quiet:  cmd.Quiet[idx],
				Opaque: cmd.Opaques[idx],
				Flags:  0,
				Key:    key,
			}
		default:
			log.Printf("[GET] Unexpected status code in HTTP response: %d\n", res.StatusCode)
			log.Printf("[GET] url: %s\n", url)
			errorOut <- common.ErrInternal
		}
	}
}

func (h *Handler) Close() error {
	// nothing to "close" here
	return nil
}

/////////////////////////////////////
// All the rest just return an error
/////////////////////////////////////
func (h *Handler) Add(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

func (h *Handler) Replace(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

func (h *Handler) Append(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

func (h *Handler) Prepend(cmd common.SetRequest) error {
	return common.ErrUnknownCmd
}

func (h *Handler) GetE(cmd common.GetRequest) (<-chan common.GetEResponse, <-chan error) {
	errchan := make(chan error, 1)
	errchan <- common.ErrUnknownCmd
	return nil, errchan
}

func (h *Handler) GAT(cmd common.GATRequest) (common.GetResponse, error) {
	return common.GetResponse{}, common.ErrUnknownCmd
}

func (h *Handler) Touch(cmd common.TouchRequest) error {
	return common.ErrUnknownCmd
}
