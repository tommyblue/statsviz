package statsviz

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arl/statsviz/websocket"
)

func TestHTTPHandlers(t *testing.T) {
	testCases := []struct {
		path        string
		handler     http.Handler
		statusCode  int
		contentType string
		resp        []byte
	}{
		{"/debug/statsviz/<script>scripty<script>", Index, http.StatusNotFound, "text/plain; charset=utf-8", []byte("404 page not found\n")},
		{"/debug/statsviz/", Index, http.StatusOK, "text/html; charset=utf-8", nil},
		{"/debug/statsviz/ws", http.HandlerFunc(Ws), http.StatusBadRequest, "text/plain; charset=utf-8", []byte("Bad Request\n")},
	}
	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tc.path, nil)
			w := httptest.NewRecorder()

			tc.handler.ServeHTTP(w, req)

			resp := w.Result()
			if got, want := resp.StatusCode, tc.statusCode; got != want {
				t.Errorf("status code: got %d; want %d", got, want)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("when reading response body, expected non-nil err; got %v", err)
			}
			if got, want := resp.Header.Get("Content-Type"), tc.contentType; got != want {
				t.Errorf("Content-Type: got %q; want %q", got, want)
			}

			if resp.StatusCode == http.StatusOK {
				return
			}
			if !bytes.Equal(body, tc.resp) {
				t.Errorf("response: got %q; want %q", body, tc.resp)
			}
		})
	}
}

var cstDialer = websocket.Dialer{
	Subprotocols:     []string{"p1", "p2"},
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 30 * time.Second,
}

func makeWsProto(s string) string {
	return "ws" + strings.TrimPrefix(s, "http")
}

func TestWSHandler(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(Ws))
	s.URL += "/debug/statsviz"
	defer s.Close()

	ws, _, err := cstDialer.Dial(makeWsProto(s.URL), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer ws.Close()

	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var stats stats
	if err := json.Unmarshal(p, &stats); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
}
