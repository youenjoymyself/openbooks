package core

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/evan-buss/openbooks/irc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServer accepts a single connection and runs script with it.
func fakeServer(t *testing.T, script func(conn net.Conn, lines <-chan string)) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		lines := make(chan string, 100)
		go func() {
			scanner := bufio.NewScanner(conn)
			for scanner.Scan() {
				lines <- scanner.Text()
			}
			close(lines)
		}()

		script(conn, lines)
	}()

	return listener.Addr().String()
}

// expect waits for the next line from the client that starts with prefix.
func expect(t *testing.T, lines <-chan string, prefix string) string {
	timeout := time.After(5 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Errorf("connection closed while waiting for %q", prefix)
				return ""
			}
			if strings.HasPrefix(line, prefix) {
				return line
			}
		case <-timeout:
			t.Errorf("timed out waiting for %q", prefix)
			return ""
		}
	}
}

func TestJoinRegistration(t *testing.T) {
	done := make(chan struct{})
	received := make(map[string]string)

	address := fakeServer(t, func(conn net.Conn, lines <-chan string) {
		defer close(done)
		expect(t, lines, "NICK tester")

		// Server requires a PONG with the correct token before continuing.
		fmt.Fprint(conn, "PING :cookie123\r\n")
		received["pong"] = expect(t, lines, "PONG")

		// Nickname is taken. Client should pick another one.
		fmt.Fprint(conn, ":server 433 * tester :Nickname is already in use.\r\n")
		received["nick"] = expect(t, lines, "NICK")

		fmt.Fprint(conn, ":Bot!b@h PRIVMSG tester_1 :\x01VERSION\x01\r\n")
		received["version"] = expect(t, lines, "NOTICE")

		fmt.Fprint(conn, ":server 001 tester_1 :Welcome\r\n")
		fmt.Fprint(conn, ":server 376 tester_1 :End of /MOTD command.\r\n")
		received["join"] = expect(t, lines, "JOIN")
	})

	conn := irc.New("tester", "OpenBooks 1.0")
	err := Join(conn, address, false)
	require.NoError(t, err)
	defer conn.Conn.Close()
	<-done

	assert.Equal(t, "PONG :cookie123", received["pong"])
	assert.Equal(t, "NICK tester_1", received["nick"])
	assert.Equal(t, "NOTICE Bot :\x01OpenBooks 1.0\x01", received["version"])
	assert.Equal(t, "JOIN #ebooks", received["join"])
	assert.Equal(t, "tester_1", conn.Nick())
}

func TestJoinNickAlwaysTaken(t *testing.T) {
	address := fakeServer(t, func(conn net.Conn, lines <-chan string) {
		for line := range lines {
			if strings.HasPrefix(line, "NICK ") {
				fmt.Fprintf(conn, ":server 433 * %s :Nickname is already in use.\r\n", strings.TrimPrefix(line, "NICK "))
			}
		}
	})

	conn := irc.New("tester", "OpenBooks 1.0")
	err := Join(conn, address, false)
	assert.ErrorIs(t, err, ErrNickInUse)
}

func TestJoinServerError(t *testing.T) {
	address := fakeServer(t, func(conn net.Conn, lines <-chan string) {
		expect(t, lines, "NICK")
		fmt.Fprint(conn, "ERROR :Closing Link: banned\r\n")
	})

	conn := irc.New("tester", "OpenBooks 1.0")
	err := Join(conn, address, false)
	assert.ErrorIs(t, err, ErrServerClosing)
}
