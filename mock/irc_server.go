package mock

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type IrcServer struct {
	Port string
	log  *log.Logger
}

func (irc *IrcServer) Start(ready chan<- struct{}) {
	irc.log = log.New(os.Stdout, "MOCK SERVER: ", 0)

	server, err := net.Listen("tcp", irc.Port)
	if err != nil {
		panic(err)
	}
	irc.log.Println("Listening on " + irc.Port)
	ready <- struct{}{}

	for {
		conn, err := server.Accept()
		if err != nil {
			panic(err)
		}
		go irc.handler(conn)
	}
}

func (irc *IrcServer) handler(conn net.Conn) {
	irc.log.Printf("Connection received from %s", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)

	nick := ""
	for scanner.Scan() {
		request := scanner.Text()
		irc.log.Printf("Request Received: %s\n", request)

		switch {
		case strings.HasPrefix(request, "NICK "):
			isFirstNick := nick == ""
			nick = strings.TrimPrefix(request, "NICK ")
			if isFirstNick {
				irc.welcome(conn, nick)
				irc.sendVersionRequest(conn, nick)
			}
		case strings.HasPrefix(request, "JOIN "):
			irc.serverHandler(conn, nick)
		case strings.HasPrefix(request, "PING "):
			fmt.Fprintf(conn, ":mock_server PONG mock_server :%s\r\n", strings.TrimPrefix(request, "PING :"))
		case strings.Contains(request, "@search"):
			go irc.searchHandler(request, conn, nick)
		case strings.Contains(request, ":!"):
			go irc.downloadHandler(request, conn, nick)
		}
	}

	irc.log.Println("Connection closed.")
}

func (irc *IrcServer) welcome(conn net.Conn, nick string) {
	fmt.Fprintf(conn, ":mock_server 001 %s :Welcome to the mock IRC server %s\r\n", nick, nick)
	fmt.Fprintf(conn, ":mock_server 376 %s :End of /MOTD command.\r\n", nick)
}

func (irc *IrcServer) sendVersionRequest(conn net.Conn, nick string) {
	irc.log.Println("Sending CTCP Version inquiry.")
	fmt.Fprintf(conn, ":mock_server PRIVMSG %s :\x01VERSION\x01\r\n", nick)
}

func (irc *IrcServer) serverHandler(conn net.Conn, nick string) {
	fmt.Fprintf(conn, ":mock_server 353 %s = #ebooks :~DV8 ~Horla +server1\r\n", nick)
	fmt.Fprintf(conn, ":mock_server 353 %s = #ebooks :~server2 %s\r\n", nick, nick)
	fmt.Fprintf(conn, ":mock_server 366 %s #ebooks :End of /NAMES list.\r\n", nick)
}

func (irc *IrcServer) searchHandler(request string, conn net.Conn, nick string) {
	irc.log.Printf("Sending search results.")
	fmt.Fprintf(conn, ":Search!Search@mock NOTICE %s :Your search for \"the great gatsby\" has been accepted. Searching...\r\n", nick)
	fmt.Fprintf(conn, ":Search!Search@mock NOTICE %s :Search returned 27 matches\r\n", nick)
	fmt.Fprintf(conn, ":SearchOok!ook@only.ook PRIVMSG %s :\x01DCC SEND SearchOok_results_for__the_great_gatsby.txt.zip 2130706433 6668 1184\x01\r\n", nick)
}

func (irc *IrcServer) downloadHandler(request string, conn net.Conn, nick string) {
	irc.log.Println("Sending book file.")
	time.Sleep(time.Second * 4)
	fmt.Fprintf(conn, ":SearchOok!ook@only.ook PRIVMSG %s :\x01DCC SEND great-gatsby.epub 2130706433 6669 358887\x01\r\n", nick)
}
