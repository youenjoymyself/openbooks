package server

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/evan-buss/openbooks/core"
)

func (server *server) NewIrcEventHandler(client *Client) core.EventHandler {
	handler := core.EventHandler{}
	handler[core.SearchResult] = client.searchResultHandler(server.config.DownloadDir)
	handler[core.BookResult] = client.bookResultHandler(server.config.DownloadDir, server.config.DisableBrowserDownloads)
	handler[core.NoResults] = client.noResultsHandler
	handler[core.BadServer] = client.badServerHandler
	handler[core.SearchAccepted] = client.searchAcceptedHandler
	handler[core.MatchesFound] = client.matchesFoundHandler
	handler[core.ServerList] = client.userListHandler(server.repository)
	handler[core.Version] = client.versionHandler(server.config.UserAgent)
	handler[core.Disconnected] = client.disconnectedHandler
	return handler
}

// searchResultHandler downloads from DCC server, parses data, and sends data to client
func (c *Client) searchResultHandler(downloadDir string) core.HandlerFunc {
	return func(text string) {
		extractedPath, err := core.DownloadExtractDCCString(filepath.Join(downloadDir, "books"), text, nil)
		if err != nil {
			c.log.Println(err)
			c.send(newSearchErrorResponse("Error when downloading search results."))
			return
		}

		bookResults, parseErrors, err := core.ParseSearchFile(extractedPath)
		if err != nil {
			c.log.Println(err)
			c.send(newSearchErrorResponse("Error when parsing search results."))
			return
		}

		if len(bookResults) == 0 && len(parseErrors) == 0 {
			c.noResultsHandler(text)
			return
		}

		// Output all errors so parser can be improved over time
		if len(parseErrors) > 0 {
			c.log.Printf("%d Search Result Parsing Errors\n", len(parseErrors))
			for _, err := range parseErrors {
				c.log.Println(err)
			}
		}

		c.log.Printf("Sending %d search results.\n", len(bookResults))
		c.send(newSearchResponse(bookResults, parseErrors))

		err = os.Remove(extractedPath)
		if err != nil {
			c.log.Printf("Error deleting search results file: %v", err)
		}
	}
}

// bookResultHandler downloads the book file and sends it over the websocket
func (c *Client) bookResultHandler(downloadDir string, disableBrowserDownloads bool) core.HandlerFunc {
	return func(text string) {
		extractedPath, err := core.DownloadExtractDCCString(filepath.Join(downloadDir, "books"), text, nil)
		if err != nil {
			c.log.Println(err)
			c.send(newDownloadErrorResponse("Error when downloading book."))
			return
		}

		c.log.Printf("Sending book entitled '%s'.\n", filepath.Base(extractedPath))
		c.send(newDownloadResponse(extractedPath, disableBrowserDownloads))
	}
}

// NoResults is called when the server returns that nothing was found for the query
func (c *Client) noResultsHandler(_ string) {
	c.send(newSearchErrorResponse("No results found for the query."))
}

// BadServer is called when the requested download fails because the server is not available
func (c *Client) badServerHandler(_ string) {
	c.send(newDownloadErrorResponse("Server is not available. Try another one."))
}

// SearchAccepted is called when the user's query is accepted into the search queue
func (c *Client) searchAcceptedHandler(_ string) {
	c.send(newStatusResponse(NOTIFY, "Search accepted into the queue."))
}

// MatchesFound is called when the server finds matches for the user's query
func (c *Client) matchesFoundHandler(num string) {
	c.send(newStatusResponse(NOTIFY, fmt.Sprintf("Found %s results for your query.", num)))
}

// disconnectedHandler is called when the IRC connection is closed unexpectedly
func (c *Client) disconnectedHandler(reason string) {
	c.send(StatusResponse{
		MessageType:      STATUS,
		NotificationType: DANGER,
		Title:            "Disconnected from the IRC server. Reload the page to reconnect.",
		Detail:           reason,
	})
}

func (c *Client) versionHandler(version string) core.HandlerFunc {
	return func(sender string) {
		c.log.Printf("Sending CTCP version response to %s", sender)
		core.SendVersionInfo(c.irc, sender, version)
	}
}

func (c *Client) userListHandler(repo *Repository) core.HandlerFunc {
	return func(text string) {
		repo.SetServers(core.ParseServers(text))
	}
}
