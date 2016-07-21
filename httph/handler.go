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
	"net/http"

	"github.com/netflix/rend/common"
	"github.com/netflix/rend/handlers"
)

type Handler struct {
	// something here
	client http.Client
}

func New() handlers.HandlerConst {
	return func() (handlers.Handler, error) {
		//lol
	}
}

func (h *Handler) Set(cmd common.SetRequest) error {
	//lol
}

func (h *Handler) Delete(cmd common.DeleteRequest) error {
	//lol
}

func (h *Handler) Get(cmd common.GetRequest) (<-chan common.GetResponse, <-chan error) {
	//lol
}

func realHandleGet(h *Handler, cmd common.GetRequest, dataOut chan common.GetResponse, errorOut chan error) {
	//lol
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
