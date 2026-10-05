package irc

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const dialTimeout = 30 * time.Second

// Strips characters that would let a caller terminate the current IRC
// command and inject another one.
var lineSanitizer = strings.NewReplacer("\r", "", "\n", "", "\x00", "")

// Conn represents an IRC connection to a server
type Conn struct {
	net.Conn
	reader   *bufio.Reader
	channel  string
	realname string

	nickMutex sync.RWMutex
	nick      string
}

// New creates a new IRC connection to the server using the supplied username and realname
func New(username, realname string) *Conn {
	irc := &Conn{
		channel:  "",
		nick:     username,
		realname: realname,
	}

	return irc
}

// Connect connects to the given server address. When TLS is enabled the
// server certificate is verified first. If verification fails, a warning is
// logged and the connection is retried without verification.
func (i *Conn) Connect(address string, enableTLS bool) error {
	conn, err := dial(address, enableTLS)
	if err != nil {
		return err
	}

	i.Conn = conn
	i.reader = bufio.NewReader(conn)

	nick := i.Nick()
	i.writeLine("USER " + nick + " 0 * :" + nick)
	i.writeLine("NICK " + nick)
	return nil
}

func dial(address string, enableTLS bool) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	if !enableTLS {
		return dialer.Dial("tcp", address)
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{})
	if err == nil || !isCertificateError(err) {
		return conn, err
	}

	log.Printf("WARNING: Unable to verify the TLS certificate of %s (%v). Falling back to an unverified TLS connection.\n", address, err)
	return tls.DialWithDialer(dialer, "tcp", address, &tls.Config{InsecureSkipVerify: true})
}

func isCertificateError(err error) bool {
	var verificationErr *tls.CertificateVerificationError
	var unknownAuthorityErr x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	var invalidErr x509.CertificateInvalidError

	return errors.As(err, &verificationErr) ||
		errors.As(err, &unknownAuthorityErr) ||
		errors.As(err, &hostnameErr) ||
		errors.As(err, &invalidErr)
}

// ReadMessage blocks until the next line is received from the server.
func (i *Conn) ReadMessage() (Message, error) {
	if !i.IsConnected() {
		return Message{}, net.ErrClosed
	}

	line, err := i.reader.ReadString('\n')
	if err != nil {
		return Message{}, err
	}

	msg := ParseMessage(line)
	i.trackNick(msg)
	return msg, nil
}

// trackNick keeps the local nickname in sync with the server. The server
// confirms our nickname in RPL_WELCOME and announces later changes with NICK.
func (i *Conn) trackNick(msg Message) {
	var nick string
	switch {
	case msg.Command == "001":
		nick = msg.Param(0)
	case msg.Command == "NICK" && strings.EqualFold(msg.Nick(), i.Nick()):
		nick = msg.Trailing()
	}

	if nick != "" {
		i.nickMutex.Lock()
		i.nick = nick
		i.nickMutex.Unlock()
	}
}

// Nick returns the nickname currently used on the server.
func (i *Conn) Nick() string {
	i.nickMutex.RLock()
	defer i.nickMutex.RUnlock()
	return i.nick
}

// SetNick requests a new nickname from the server.
func (i *Conn) SetNick(nick string) {
	i.nickMutex.Lock()
	i.nick = nick
	i.nickMutex.Unlock()
	i.writeLine("NICK " + nick)
}

// Channel returns the name of the joined channel without the leading '#'.
func (i *Conn) Channel() string {
	return i.channel
}

// Realname returns the realname / version string the connection was created with.
func (i *Conn) Realname() string {
	return i.realname
}

// Disconnect closes connection to the IRC server
func (i *Conn) Disconnect() {
	if !i.IsConnected() {
		return
	}
	i.writeLine("QUIT :Goodbye")
	i.Conn.Close()
}

// SendMessage sends the given message string to the connected IRC server
func (i *Conn) SendMessage(message string) {
	i.writeLine("PRIVMSG #" + i.channel + " :" + message)
}

// SendNotice sends a notice message to the specified user
func (i *Conn) SendNotice(user string, message string) {
	i.writeLine("NOTICE " + user + " :" + message)
}

// JoinChannel joins the channel given by channel string
func (i *Conn) JoinChannel(channel string) {
	i.channel = channel
	i.writeLine("JOIN #" + channel)
}

// GetUsers sends a NAMES request to the IRC server
func (i *Conn) GetUsers(channel string) {
	i.writeLine("NAMES #" + channel)
}

// Pong sends a Pong message to the server in response to a PING request
func (i *Conn) Pong(token string) {
	i.writeLine("PONG :" + token)
}

// IsConnected returns true if the IRC connection is not null
func (i *Conn) IsConnected() bool {
	return i.Conn != nil
}

func (i *Conn) writeLine(line string) {
	if !i.IsConnected() {
		return
	}
	i.Write([]byte(lineSanitizer.Replace(line) + "\r\n"))
}
