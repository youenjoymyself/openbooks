package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/evan-buss/openbooks/core"
	"github.com/evan-buss/openbooks/util"
)

// RequestHandler defines a generic handle() method that is called when a specific request type is made
type RequestHandler interface {
	handle(c *Client)
}

// messageRouter is used to parse the incoming request and respond appropriately
func (server *server) routeMessage(message Request, c *Client) {
	switch message.MessageType {
	case CONNECT:
		c.startIrcConnection(server)
	case SEARCH:
		var request SearchRequest
		if !server.decodePayload(message, &request, c) {
			return
		}
		c.sendSearchRequest(&request, server)
	case DOWNLOAD:
		var request DownloadRequest
		if !server.decodePayload(message, &request, c) {
			return
		}
		c.sendDownloadRequest(&request)
	default:
		server.log.Println("Unknown request type received.")
	}
}

// decodePayload unmarshals the request payload into target. Sends an error to
// the client and returns false if the payload is invalid.
func (server *server) decodePayload(message Request, target interface{}, c *Client) bool {
	err := json.Unmarshal(message.Payload, target)
	if err != nil {
		server.log.Printf("Invalid request payload. %s.\n", err.Error())
		c.send(newErrorResponse("Unknown request payload."))
		return false
	}
	return true
}

// handle ConnectionRequests and either connect to the server or do nothing
func (c *Client) startIrcConnection(server *server) {
	if !c.ircConnecting.CompareAndSwap(false, true) {
		c.log.Println("Ignoring connection request. Already connected to IRC.")
		return
	}

	err := core.Join(c.irc, server.config.Server, server.config.EnableTLS)
	if err != nil {
		c.ircConnecting.Store(false)
		c.log.Println(err)
		c.send(newErrorResponse(fmt.Sprintf("Unable to connect to IRC server. %s", err)))
		return
	}

	handler := server.NewIrcEventHandler(c)

	if server.config.Log {
		logger, _, err := util.CreateLogFile(c.irc.Nick(), server.config.DownloadDir)
		if err != nil {
			server.log.Println(err)
		} else {
			handler[core.Message] = func(text string) { logger.Println(text) }
		}
	}

	go core.StartReader(c.ctx, c.irc, handler)

	c.send(ConnectionResponse{
		StatusResponse: StatusResponse{
			MessageType:      CONNECT,
			NotificationType: SUCCESS,
			Title:            "Welcome, connection established.",
			Detail:           fmt.Sprintf("IRC username %s", c.irc.Nick()),
		},
		Name: c.irc.Nick(),
	})
}

// handle SearchRequests and send the query to the book server
func (c *Client) sendSearchRequest(s *SearchRequest, server *server) {
	server.lastSearchMutex.Lock()
	defer server.lastSearchMutex.Unlock()

	nextAvailableSearch := server.lastSearch.Add(server.config.SearchTimeout)

	if time.Now().Before(nextAvailableSearch) {
		remainingSeconds := time.Until(nextAvailableSearch).Seconds()
		c.send(newRateLimitResponse(remainingSeconds))

		return
	}

	core.SearchBook(c.irc, server.config.SearchBot, s.Query)
	server.lastSearch = time.Now()

	c.send(newStatusResponse(NOTIFY, "Search request sent."))
}

// handle DownloadRequests by sending the request to the book server
func (c *Client) sendDownloadRequest(d *DownloadRequest) {
	core.DownloadBook(c.irc, d.Book)
	c.send(newStatusResponse(NOTIFY, "Download request received."))
}
