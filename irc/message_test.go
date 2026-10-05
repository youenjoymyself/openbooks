package irc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessage(t *testing.T) {
	cases := []struct {
		line    string
		prefix  string
		command string
		params  []string
	}{
		{"PING :irc.example.net\r\n", "", "PING", []string{"irc.example.net"}},
		{":nick!user@host PRIVMSG #ebooks :hello there", "nick!user@host", "PRIVMSG", []string{"#ebooks", "hello there"}},
		{":server 353 me = #ebooks :~DV8 +Horla evan", "server", "353", []string{"me", "=", "#ebooks", "~DV8 +Horla evan"}},
		{":server 366 me #ebooks :End of /NAMES list.", "server", "366", []string{"me", "#ebooks", "End of /NAMES list."}},
		{"@time=2024-01-01T00:00:00Z :nick!u@h NOTICE me :text", "nick!u@h", "NOTICE", []string{"me", "text"}},
		{":server 001 me :Welcome", "server", "001", []string{"me", "Welcome"}},
		{"privmsg  target   :spaced  out ", "", "PRIVMSG", []string{"target", "spaced  out "}},
		{"", "", "", nil},
		{":onlyprefix", "onlyprefix", "", nil},
	}

	for _, c := range cases {
		msg := ParseMessage(c.line)
		assert.Equal(t, c.prefix, msg.Prefix, c.line)
		assert.Equal(t, c.command, msg.Command, c.line)
		assert.Equal(t, c.params, msg.Params, c.line)
	}
}

func TestMessageHelpers(t *testing.T) {
	msg := ParseMessage(":Search!Search@host PRIVMSG me :\x01DCC SEND file.zip 1 2 3\x01")
	assert.Equal(t, "Search", msg.Nick())
	assert.Equal(t, "me", msg.Param(0))
	assert.Equal(t, "", msg.Param(5))
	body, ok := msg.CTCP()
	assert.True(t, ok)
	assert.Equal(t, "DCC SEND file.zip 1 2 3", body)

	_, ok = ParseMessage(":a!b@c PRIVMSG me :DCC SEND file.zip 1 2 3").CTCP()
	assert.False(t, ok, "plain text must not be treated as CTCP")

	assert.Equal(t, "", Message{}.Trailing())
}
