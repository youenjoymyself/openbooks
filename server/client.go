package server

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/evan-buss/openbooks/irc"
	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// Unique ID for the client
	uuid uuid.UUID

	// The websocket connection.
	conn *websocket.Conn

	// Messages to send to the client ws connection. Use send() to add to it.
	outbox chan interface{}

	// Individual IRC connection per connected client.
	irc *irc.Conn

	// Set once the client has started connecting to IRC.
	ircConnecting atomic.Bool

	log *log.Logger

	// Context is used to signal when this client should close.
	ctx    context.Context
	cancel context.CancelFunc
}

// send queues a message for the websocket connection. It is safe to call
// after the client has disconnected, in which case the message is dropped.
func (c *Client) send(message interface{}) {
	select {
	case c.outbox <- message:
	case <-c.ctx.Done():
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (server *server) readPump(c *Client) {
	defer func() {
		// Cancel first so the IRC reader knows the disconnect was intentional.
		c.cancel()
		c.irc.Disconnect()
		c.conn.Close()
		server.removeClient(c)
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var request Request
			err := c.conn.ReadJSON(&request)

			if err != nil {
				c.log.Printf("Connection Closed: %v", err)
				return
			}

			c.log.Printf("%s Message Received\n", request.MessageType)

			server.routeMessage(request, c)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (server *server) writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		// Unblocks readPump so that it cleans up the client.
		c.conn.Close()
	}()

	for {
		select {
		case message := <-c.outbox:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteJSON(message)
			if err != nil {
				c.log.Printf("Error writing JSON to websocket: %s\n", err)
				return
			}
		case <-c.ctx.Done():
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
