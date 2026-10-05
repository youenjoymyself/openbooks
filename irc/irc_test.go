package irc

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureLog(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(original) })
	return &buf
}

// Messages containing line breaks must not be able to send additional commands.
func TestCommandInjection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	lines := make(chan string, 10)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	conn := New("tester", "OpenBooks test")
	require.NoError(t, conn.Connect(listener.Addr().String(), false))
	conn.JoinChannel("ebooks")
	conn.SendMessage("@search query\r\nQUIT :injected\nPRIVMSG #other :spam")
	conn.Conn.Close()

	var received []string
	for line := range lines {
		received = append(received, line)
	}

	assert.Equal(t, []string{
		"USER tester 0 * :tester",
		"NICK tester",
		"JOIN #ebooks",
		"PRIVMSG #ebooks :@search queryQUIT :injectedPRIVMSG #other :spam",
	}, received)
}

// Servers with self-signed certificates (like irc.irchighway.net) are still
// reachable, but a warning is logged.
func TestTLSFallbackOnCertificateError(t *testing.T) {
	server := httptest.NewTLSServer(http.NotFoundHandler())
	defer server.Close()
	logs := captureLog(t)

	conn := New("tester", "OpenBooks test")
	err := conn.Connect(server.Listener.Addr().String(), true)
	require.NoError(t, err)
	defer conn.Conn.Close()

	assert.Contains(t, logs.String(), "Falling back to an unverified TLS connection")
}

// Network errors are returned as is and don't trigger the insecure fallback.
func TestNoTLSFallbackOnNetworkError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	listener.Close()
	logs := captureLog(t)

	conn := New("tester", "OpenBooks test")
	err = conn.Connect(address, true)
	assert.Error(t, err)
	assert.False(t, strings.Contains(logs.String(), "Falling back"))
}

func TestNickTracking(t *testing.T) {
	conn := New("tester", "")
	conn.trackNick(ParseMessage(":server 001 tester_1 :Welcome"))
	assert.Equal(t, "tester_1", conn.Nick())

	conn.trackNick(ParseMessage(":someone!u@h NICK :other"))
	assert.Equal(t, "tester_1", conn.Nick(), "other users changing nick must be ignored")

	conn.trackNick(ParseMessage(":tester_1!u@h NICK :renamed"))
	assert.Equal(t, "renamed", conn.Nick())
}
