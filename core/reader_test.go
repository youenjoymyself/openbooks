package core

import (
	"strings"
	"testing"

	"github.com/evan-buss/openbooks/irc"
	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	const nick = "me"
	const channel = "#ebooks"

	cases := []struct {
		reason string
		line   string
		event  event
		text   string
	}{
		{
			"search results sent to us",
			":Search!s@h PRIVMSG me :\x01DCC SEND SearchBot_results_for__gatsby.txt.zip 2130706433 6668 1184\x01",
			SearchResult, "DCC SEND SearchBot_results_for__gatsby.txt.zip 2130706433 6668 1184",
		},
		{
			"book sent to us, nick is case insensitive",
			":DV8!d@h PRIVMSG Me :\x01DCC SEND book.epub 2130706433 6669 100\x01",
			BookResult, "DCC SEND book.epub 2130706433 6669 100",
		},
		{
			"DCC SEND in the channel must be ignored",
			":attacker!a@h PRIVMSG #ebooks :\x01DCC SEND ../../.bashrc 2130706433 6669 100\x01",
			noOp, "",
		},
		{
			"DCC SEND to another user must be ignored",
			":attacker!a@h PRIVMSG someone :\x01DCC SEND book.epub 2130706433 6669 100\x01",
			noOp, "",
		},
		{
			"DCC SEND text that isn't CTCP must be ignored",
			":attacker!a@h PRIVMSG me :DCC SEND book.epub 2130706433 6669 100",
			noOp, "",
		},
		{
			"version request",
			":Bot!b@h PRIVMSG me :\x01VERSION\x01",
			Version, "Bot",
		},
		{
			"no results",
			":Search!s@h NOTICE me :Sorry, your search for \"xyz\" returned no matches.",
			NoResults, "Sorry, your search for \"xyz\" returned no matches.",
		},
		{
			"server unavailable",
			":DV8!d@h NOTICE me :Server is not available, try another server",
			BadServer, "Server is not available, try another server",
		},
		{
			"search accepted",
			":Search!s@h NOTICE me :Your search has been accepted. Searching...",
			SearchAccepted, "Your search has been accepted. Searching...",
		},
		{
			"number of matches",
			":Search!s@h NOTICE me :Search returned 27 matches for \"gatsby\"",
			MatchesFound, "27",
		},
		{
			"malformed matches notice must not panic",
			":troll!t@h NOTICE me :matches returned",
			noOp, "",
		},
		{
			"notices to the channel are ignored",
			":troll!t@h NOTICE #ebooks :Sorry everyone",
			noOp, "",
		},
		{
			"chat text containing numerics is ignored",
			":user!u@h PRIVMSG #ebooks :353 366 PING",
			noOp, "",
		},
	}

	for _, c := range cases {
		var users strings.Builder
		event, text := classify(irc.ParseMessage(c.line), nick, channel, &users)
		assert.Equal(t, c.event, event, c.reason)
		assert.Equal(t, c.text, text, c.reason)
	}
}

// The server list is split across multiple 353 lines. Every nick on every
// line must be parsed.
func TestClassifyNamesList(t *testing.T) {
	lines := []string{
		":server 353 me = #ebooks :~Oatmeal +DV8 regular",
		":server 353 me = #other :+NotInEbooks",
		":server 353 me = #ebooks :@Horla me +peapod",
	}

	var users strings.Builder
	for _, line := range lines {
		event, _ := classify(irc.ParseMessage(line), "me", "#ebooks", &users)
		assert.Equal(t, noOp, event)
	}

	event, text := classify(irc.ParseMessage(":server 366 me #ebooks :End of /NAMES list."), "me", "#ebooks", &users)
	assert.Equal(t, ServerList, event)

	servers := ParseServers(text)
	assert.Equal(t, []string{"DV8", "Horla", "Oatmeal", "peapod"}, servers.ElevatedUsers)
	assert.Equal(t, []string{"me", "regular"}, servers.RegularUsers)
	assert.Empty(t, users.String(), "builder is reset after the list ends")
}
