package core

import (
	"context"
	"log"
	"regexp"
	"strings"

	"github.com/evan-buss/openbooks/irc"
)

type event int

const (
	noOp           = event(0)
	Message        = event(1)
	SearchResult   = event(2)
	BookResult     = event(3)
	NoResults      = event(4)
	BadServer      = event(5)
	SearchAccepted = event(6)
	MatchesFound   = event(7)
	ServerList     = event(8)
	Version        = event(10)
	Disconnected   = event(11)
)

// Unique identifiers found in the message for various different events.
const (
	sendMessage            = "DCC SEND "
	noResults              = "Sorry"
	serverUnavailable      = "try another server"
	searchAccepted         = "has been accepted"
	searchResultIdentifier = "_results_for"
	versionInquiry         = "VERSION"
	namesReply             = "353"
	endOfNames             = "366"
)

var matchesRegex = regexp.MustCompile(`returned\s+(\S+)\s+matches`)

type HandlerFunc func(text string)
type EventHandler map[event]HandlerFunc

// StartReader reads messages from the IRC connection until it is closed and
// invokes the matching handler for each event. The Disconnected handler is
// called when the connection drops, unless ctx has already been cancelled.
func StartReader(ctx context.Context, conn *irc.Conn, handler EventHandler) {
	var users strings.Builder

	for {
		msg, err := conn.ReadMessage()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("IRC connection closed: %v\n", err)
			if invoke, ok := handler[Disconnected]; ok {
				invoke(err.Error())
			}
			return
		}

		// Send raw message if they want to recieve it (logging purposes)
		if invoke, ok := handler[Message]; ok {
			invoke(strings.TrimRight(msg.Raw, "\r\n"))
		}

		if msg.Command == "PING" {
			conn.Pong(msg.Trailing())
			continue
		}

		event, text := classify(msg, conn.Nick(), "#"+conn.Channel(), &users)
		if invoke, ok := handler[event]; ok {
			go invoke(text)
		}
	}
}

// classify determines which event a message represents and the text that
// should be passed to its handler. users accumulates NAMES replies until the
// end of the list is received.
func classify(msg irc.Message, nick, channel string, users *strings.Builder) (event, string) {
	addressedToUs := strings.EqualFold(msg.Param(0), nick)

	switch msg.Command {
	case "PRIVMSG":
		body, isCTCP := msg.CTCP()
		if !addressedToUs || !isCTCP {
			return noOp, ""
		}
		if strings.HasPrefix(body, sendMessage) {
			if strings.Contains(body, searchResultIdentifier) {
				return SearchResult, body
			}
			return BookResult, body
		}
		if body == versionInquiry {
			return Version, msg.Nick()
		}
	case "NOTICE":
		if !addressedToUs {
			return noOp, ""
		}
		text := msg.Trailing()
		switch {
		case strings.Contains(text, noResults):
			return NoResults, text
		case strings.Contains(text, serverUnavailable):
			return BadServer, text
		case strings.Contains(text, searchAccepted):
			return SearchAccepted, text
		}
		if groups := matchesRegex.FindStringSubmatch(text); groups != nil {
			return MatchesFound, groups[1]
		}
	case namesReply:
		if strings.EqualFold(msg.Param(2), channel) {
			users.WriteString(msg.Trailing())
			users.WriteString(" ")
		}
	case endOfNames:
		if strings.EqualFold(msg.Param(1), channel) {
			text := users.String()
			users.Reset()
			return ServerList, text
		}
	}

	return noOp, ""
}
