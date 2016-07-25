package httph_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/netflix/rend-http/httph"
	"github.com/netflix/rend/common"
	"github.com/netflix/rend/handlers"
)

type server struct {
	data      map[string]string
	senderror bool
}

func newServer(senderror bool) *server {
	return &server{
		data:      make(map[string]string),
		senderror: senderror,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := strings.TrimPrefix(req.URL.Path, "/evcrest/v1.0/evcache/")

	if s.senderror {
		w.WriteHeader(500)
		return
	}

	switch req.Method {
	case "GET":
		if data, ok := s.data[key]; ok {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte(data))
		} else {
			w.WriteHeader(404)
		}

	case "POST":
		fallthrough
	case "PUT":
		// verify ttl
		ttl := req.URL.Query().Get("ttl")
		if ttl == "" {
			w.WriteHeader(400)
		}
		if _, err := strconv.Atoi(ttl); err != nil {
			w.WriteHeader(400)
		}

		// Don't bother with TTL here for testing
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(500)
		}
		s.data[key] = string(data)
		w.WriteHeader(202)

	case "DELETE":
		delete(s.data, key)
		w.WriteHeader(200)
	}
}

func handlerFromTestServer(ts *httptest.Server) (handlers.Handler, error) {
	hostAndPort := strings.TrimPrefix(ts.URL, "http://")
	parts := strings.Split(hostAndPort, ":")

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}

	return httph.New(host, port, "evcache")()
}

func TestGetHit(t *testing.T) {
	s := newServer(false)
	ts := httptest.NewServer(s)
	defer ts.Close()

	handler, err := handlerFromTestServer(ts)
	if err != nil {
		t.Errorf("Handler creation failed: %s", err.Error())
	}

	// somewhat dirty, but I don't mind
	s.data["foo"] = "bar"

	datchan, errchan := handler.Get(common.GetRequest{
		Keys:    [][]byte{[]byte("foo")},
		Opaques: []uint32{0},
		Quiet:   []bool{false},
	})

	select {
	case res := <-datchan:
		if !res.Miss {
			t.Log("Success, got item")
			t.Logf("Value: %s", string(res.Data))
		} else {
			t.Error("Response was a miss")
		}
	case err := <-errchan:
		t.Errorf("Failed to retrieve item: %s", err.Error())
	}
}

func TestGetMiss(t *testing.T) {
	s := newServer(false)
	ts := httptest.NewServer(s)
	defer ts.Close()

	handler, err := handlerFromTestServer(ts)
	if err != nil {
		t.Errorf("Handler creation failed: %s", err.Error())
	}

	datchan, errchan := handler.Get(common.GetRequest{
		Keys:    [][]byte{[]byte("foo")},
		Opaques: []uint32{0},
		Quiet:   []bool{false},
	})

	select {
	case res := <-datchan:
		if res.Miss {
			t.Log("Success, miss")
			t.Logf("Value: %s", string(res.Data))
		} else {
			t.Error("Response was a hit")
		}
	case err := <-errchan:
		t.Errorf("Failed to retrieve item: %s", err.Error())
	}
}

func TestGetError(t *testing.T) {
	s := newServer(true)
	ts := httptest.NewServer(s)
	defer ts.Close()

	handler, err := handlerFromTestServer(ts)
	if err != nil {
		t.Errorf("Handler creation failed: %s", err.Error())
	}

	datchan, errchan := handler.Get(common.GetRequest{
		Keys:    [][]byte{[]byte("foo")},
		Opaques: []uint32{0},
		Quiet:   []bool{false},
	})

	select {
	case res := <-datchan:
		t.Errorf("Should have received an error.\nResponse: %#v", res)
	case err := <-errchan:
		t.Logf("Properly received error: %s", err.Error())
	}
}
