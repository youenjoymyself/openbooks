package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evan-buss/openbooks/irc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, config Config) *server {
	if config.DownloadDir == "" {
		config.DownloadDir = t.TempDir()
	}
	if config.Basepath == "" {
		config.Basepath = "/"
	}
	require.NoError(t, os.MkdirAll(filepath.Join(config.DownloadDir, "books"), 0755))

	s := New(config)
	s.log = log.New(io.Discard, "", 0)
	return s
}

func testClient() *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		uuid:   uuid.New(),
		outbox: make(chan interface{}, 10),
		irc:    irc.New("tester", "OpenBooks test"),
		log:    log.New(io.Discard, "", 0),
		ctx:    ctx,
		cancel: cancel,
	}
}

func deleteRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.AddCookie(&http.Cookie{Name: "OpenBooks", Value: uuid.New().String()})
	return req
}

func TestDeleteBook(t *testing.T) {
	s := testServer(t, Config{Persist: true})
	book := filepath.Join(s.config.DownloadDir, "books", "My Book.epub")
	require.NoError(t, os.WriteFile(book, []byte("book"), 0644))

	recorder := httptest.NewRecorder()
	s.registerRoutes().ServeHTTP(recorder, deleteRequest("/library/My%20Book.epub"))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NoFileExists(t, book)
}

func TestDeleteBookPathTraversal(t *testing.T) {
	s := testServer(t, Config{Persist: true})
	secret := filepath.Join(s.config.DownloadDir, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0644))

	for _, path := range []string{
		"/library/..%2Fsecret.txt",
		"/library/..%5Csecret.txt",
		"/library/..",
	} {
		recorder := httptest.NewRecorder()
		s.registerRoutes().ServeHTTP(recorder, deleteRequest(path))

		assert.NotEqual(t, http.StatusOK, recorder.Code, path)
		assert.FileExists(t, secret, path)
	}
}

func TestCheckOrigin(t *testing.T) {
	s := testServer(t, Config{AllowedOrigins: []string{"https://books.example.com/"}})

	cases := []struct {
		origin        string
		host          string
		forwardedHost string
		allowed       bool
	}{
		{"", "localhost:5228", "", true},
		{"http://localhost:5228", "localhost:5228", "", true},
		{"http://192.168.1.10:8080", "192.168.1.10:8080", "", true},
		{"https://books.home.lan", "openbooks:80", "books.home.lan", true},
		{"https://books.example.com", "openbooks:80", "", true},
		{"http://127.0.0.1:5173", "127.0.0.1:5228", "", true},
		{"https://evil.example.com", "localhost:5228", "", false},
		{"http://localhost:5228.evil.com", "localhost:5228", "", false},
		{"null", "localhost:5228", "", false},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.Host = c.host
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		if c.forwardedHost != "" {
			req.Header.Set("X-Forwarded-Host", c.forwardedHost)
		}
		assert.Equal(t, c.allowed, s.checkOrigin(req), c.origin)
	}
}

// Malformed websocket requests must not crash the server.
func TestRouteMessageInvalidPayload(t *testing.T) {
	s := testServer(t, Config{SearchTimeout: time.Second})
	c := testClient()

	assert.NotPanics(t, func() {
		s.routeMessage(Request{MessageType: SEARCH, Payload: json.RawMessage("null")}, c)
		s.routeMessage(Request{MessageType: DOWNLOAD, Payload: json.RawMessage("null")}, c)
		s.routeMessage(Request{MessageType: SEARCH, Payload: json.RawMessage(`"text"`)}, c)
		s.routeMessage(Request{MessageType: DOWNLOAD}, c)
		s.routeMessage(Request{MessageType: 99}, c)
	})

	var titles []string
	for len(c.outbox) > 0 {
		titles = append(titles, (<-c.outbox).(StatusResponse).Title)
	}
	assert.Contains(t, titles, "Unknown request payload.")
}

// IRC handlers keep running after the browser disconnects. Sending to a
// disconnected client must not block or panic.
func TestSendAfterDisconnect(t *testing.T) {
	s := testServer(t, Config{})
	c := testClient()
	c.outbox = make(chan interface{}) // unbuffered, nobody reading
	require.True(t, s.addClient(c))

	s.removeClient(c)

	done := make(chan struct{})
	go func() {
		c.send(newErrorResponse("late message"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("send blocked after the client disconnected")
	}
	assert.True(t, s.canConnect(uuid.New()), "client was removed")
}

func TestSingleClient(t *testing.T) {
	s := testServer(t, Config{})
	first, second := testClient(), testClient()

	assert.True(t, s.addClient(first))
	assert.False(t, s.canConnect(second.uuid))
	assert.False(t, s.addClient(second))

	s.removeClient(first)
	assert.True(t, s.addClient(second))
}

// Failures are sent under the type of the request that failed, so the client
// can end its pending search or download.
func TestFailureResponses(t *testing.T) {
	search, err := json.Marshal(newSearchErrorResponse("No results found for the query."))
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":2,"appearance":3,"title":"No results found for the query.","detail":"","books":[],"errors":[]}`, string(search))

	download := newDownloadErrorResponse("Server is not available. Try another one.")
	assert.Equal(t, DOWNLOAD, download.MessageType)
	assert.Equal(t, DANGER, download.NotificationType)
	assert.Empty(t, download.DownloadPath)
}
