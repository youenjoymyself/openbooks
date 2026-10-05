package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evan-buss/openbooks/irc"
)

// Specific irc.irchighway.net commands

const (
	// Maximum time to wait for the server to accept our connection.
	registrationTimeout = 60 * time.Second
	// Number of alternative nicknames to try if ours is taken.
	maxNickRetries = 5
)

var (
	ErrNickInUse     = errors.New("nickname is already in use")
	ErrNickInvalid   = errors.New("nickname was rejected by the server")
	ErrServerClosing = errors.New("server closed the connection")
)

// Join connects to the irc.irchighway.net server and joins the #ebooks channel
func Join(conn *irc.Conn, address string, enableTLS bool) error {
	err := conn.Connect(address, enableTLS)
	if err != nil {
		return err
	}

	err = waitForRegistration(conn)
	if err != nil {
		conn.Disconnect()
		return err
	}

	conn.JoinChannel("ebooks")
	return nil
}

// waitForRegistration reads server messages until the connection is fully
// registered (end of the MOTD). Picks an alternative nickname if ours is taken.
func waitForRegistration(conn *irc.Conn) error {
	conn.SetReadDeadline(time.Now().Add(registrationTimeout))
	defer conn.SetReadDeadline(time.Time{})

	baseNick := conn.Nick()
	retries := 0
	welcomed := false

	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("unable to register with irc server: %w", err)
		}

		switch msg.Command {
		case "PING":
			conn.Pong(msg.Trailing())
		case "PRIVMSG":
			if body, ok := msg.CTCP(); ok && body == versionInquiry {
				SendVersionInfo(conn, msg.Nick(), conn.Realname())
			}
		case "001": // RPL_WELCOME
			welcomed = true
		case "376", "422": // RPL_ENDOFMOTD, ERR_NOMOTD
			if welcomed {
				return nil
			}
		case "433", "436": // ERR_NICKNAMEINUSE, ERR_NICKCOLLISION
			if welcomed {
				continue
			}
			retries++
			if retries > maxNickRetries {
				return fmt.Errorf("%w: %s", ErrNickInUse, baseNick)
			}
			conn.SetNick(fmt.Sprintf("%s_%d", baseNick, retries))
		case "432": // ERR_ERRONEUSNICKNAME
			return fmt.Errorf("%w: %s", ErrNickInvalid, strings.TrimSpace(msg.Trailing()))
		case "ERROR":
			return fmt.Errorf("%w: %s", ErrServerClosing, msg.Trailing())
		}
	}
}

// SearchBook sends a search query to the search bot
func SearchBook(irc *irc.Conn, searchBot string, query string) {
	searchBot = strings.TrimPrefix(searchBot, "@")
	irc.SendMessage(fmt.Sprintf("@%s %s", searchBot, query))
}

// DownloadBook sends the book string to the download bot
func DownloadBook(irc *irc.Conn, book string) {
	irc.SendMessage(book)
}

// SendVersionInfo sends a CTCP Version response to the given nickname.
func SendVersionInfo(irc *irc.Conn, sender string, version string) {
	// TODO: Figure out if there's an automated way to adjust this...
	irc.SendNotice(sender, fmt.Sprintf("\x01%s\x01", version))
}
