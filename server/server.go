package server

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/cors"
)

type server struct {
	// Shared app configuration
	config *Config

	// Shared data
	repository *Repository

	// Registered clients. Guarded by clientsMutex.
	clients      map[uuid.UUID]*Client
	clientsMutex sync.RWMutex

	log *log.Logger

	// Mutex to guard the lastSearch timestamp
	lastSearchMutex sync.Mutex

	// The time the last search was performed. Used to rate limit searches.
	lastSearch time.Time
}

// Config contains settings for server
type Config struct {
	Log                     bool
	Host                    string
	Port                    string
	UserName                string
	Persist                 bool
	DownloadDir             string
	Basepath                string
	Server                  string
	EnableTLS               bool
	SearchTimeout           time.Duration
	SearchBot               string
	DisableBrowserDownloads bool
	UserAgent               string
	// Additional origins (ex. "https://books.example.com") allowed to open a
	// websocket connection. The server's own host is always allowed.
	AllowedOrigins []string
}

func New(config Config) *server {
	return &server{
		repository: NewRepository(),
		config:     &config,
		clients:    make(map[uuid.UUID]*Client),
		log:        log.New(os.Stdout, "SERVER: ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Start instantiates the web server and opens the browser
func Start(config Config) {
	createBooksDirectory(config)
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	corsConfig := cors.Options{
		AllowCredentials: true,
		AllowedOrigins:   []string{"http://127.0.0.1:5173"},
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "DELETE"},
	}
	router.Use(cors.New(corsConfig).Handler)

	server := New(config)
	routes := server.registerRoutes()

	server.registerGracefulShutdown()
	router.Mount(config.Basepath, routes)

	server.log.Printf("Base Path: %s\n", config.Basepath)
	server.log.Printf("OpenBooks is listening on %s", net.JoinHostPort(config.Host, config.Port))
	server.log.Printf("Download Directory: %s\n", config.DownloadDir)
	server.log.Printf("Open http://localhost:%v%s in your browser.", config.Port, config.Basepath)
	server.log.Fatal(http.ListenAndServe(net.JoinHostPort(config.Host, config.Port), router))
}

// addClient registers a new client. Only one client may be connected at a
// time, so it returns false if another client is already registered.
func (server *server) addClient(client *Client) bool {
	server.clientsMutex.Lock()
	defer server.clientsMutex.Unlock()

	if len(server.clients) > 0 {
		return false
	}
	server.clients[client.uuid] = client
	return true
}

// removeClient unregisters a client and signals its goroutines to stop.
func (server *server) removeClient(client *Client) {
	server.clientsMutex.Lock()
	defer server.clientsMutex.Unlock()

	client.cancel()
	if server.clients[client.uuid] == client {
		delete(server.clients, client.uuid)
	}
}

// canConnect returns false if the user is already connected or another user is
// connected.
func (server *server) canConnect(userId uuid.UUID) bool {
	server.clientsMutex.RLock()
	defer server.clientsMutex.RUnlock()

	_, alreadyConnected := server.clients[userId]
	return !alreadyConnected && len(server.clients) == 0
}

func (server *server) registerGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		server.log.Println("Graceful shutdown.")
		// Cancel each client. Triggering all reader/writer WS handlers to close.
		server.clientsMutex.RLock()
		for _, client := range server.clients {
			client.irc.Disconnect()
			client.cancel()
		}
		server.clientsMutex.RUnlock()
		time.Sleep(time.Second)
		os.Exit(0)
	}()
}

func createBooksDirectory(config Config) {
	err := os.MkdirAll(filepath.Join(config.DownloadDir, "books"), os.FileMode(0755))
	if err != nil {
		panic(err)
	}
}
