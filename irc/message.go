package irc

import (
	"strings"
)

// Message is a single parsed IRC protocol line.
// Format: [@tags] [:prefix] COMMAND [params...] [:trailing]
type Message struct {
	Raw     string
	Prefix  string
	Command string
	Params  []string
}

// ParseMessage parses a raw IRC line into its parts. It never fails; lines
// that don't follow the protocol produce a Message with an empty Command.
func ParseMessage(line string) Message {
	msg := Message{Raw: line}
	rest := strings.TrimRight(line, "\r\n")

	// IRCv3 message tags are ignored.
	if strings.HasPrefix(rest, "@") {
		_, rest, _ = strings.Cut(rest, " ")
		rest = strings.TrimLeft(rest, " ")
	}

	if strings.HasPrefix(rest, ":") {
		msg.Prefix, rest, _ = strings.Cut(rest[1:], " ")
		rest = strings.TrimLeft(rest, " ")
	}

	msg.Command, rest, _ = strings.Cut(rest, " ")
	msg.Command = strings.ToUpper(msg.Command)

	for rest != "" {
		rest = strings.TrimLeft(rest, " ")
		if rest == "" {
			break
		}
		if strings.HasPrefix(rest, ":") {
			msg.Params = append(msg.Params, rest[1:])
			break
		}
		var param string
		param, rest, _ = strings.Cut(rest, " ")
		msg.Params = append(msg.Params, param)
	}

	return msg
}

// Param returns the i-th parameter or an empty string if it doesn't exist.
func (m Message) Param(i int) string {
	if i < 0 || i >= len(m.Params) {
		return ""
	}
	return m.Params[i]
}

// Trailing returns the last parameter, which holds the message text for
// PRIVMSG and NOTICE.
func (m Message) Trailing() string {
	return m.Param(len(m.Params) - 1)
}

// Nick returns the nickname portion of the message prefix.
func (m Message) Nick() string {
	nick, _, _ := strings.Cut(m.Prefix, "!")
	return nick
}

// CTCP returns the body of a CTCP request (text wrapped in \x01) and true, or
// an empty string and false if the trailing parameter is not a CTCP message.
func (m Message) CTCP() (string, bool) {
	text := m.Trailing()
	if len(text) < 2 || text[0] != '\x01' {
		return "", false
	}
	return strings.TrimSuffix(text[1:], "\x01"), true
}
