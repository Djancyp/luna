package luna

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type HotReload struct {
	engine           *Engine
	logger           zerolog.Logger
	connectedClients map[string][]*websocket.Conn
	mu               sync.Mutex // Mutex for safe access to connectedClients
}

// newHotReload creates a new HotReload instance
func newHotReload(engine *Engine) *HotReload {
	return &HotReload{
		engine:           engine,
		logger:           engine.Logger,
		connectedClients: make(map[string][]*websocket.Conn),
	}
}

// Start starts the hot reload server and watcher
func (hr *HotReload) Start(basedir string) {
	go hr.startServer()
	go hr.startWatcher(basedir)
}

// startServer starts the hot reload websocket server
func (hr *HotReload) startServer() {
	hr.logger.Info().Msgf("Hot reload websocket running on port %d", hr.engine.Config.HotReloadServerPort)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			hr.logger.Err(err).Msg("Failed to upgrade websocket")
			return
		}
		defer ws.Close() // Ensure connection is closed when the function exits

		_, routeID, err := ws.ReadMessage()
		if err != nil {
			hr.logger.Err(err).Msg("Failed to read routeID from websocket")
			return
		}
		err = ws.WriteMessage(websocket.TextMessage, []byte("Connected"))
		if err != nil {
			hr.logger.Err(err).Msg("Failed to send 'Connected' message")
			return
		}

		// Add client to connectedClients in a thread-safe manner
		hr.mu.Lock()
		hr.connectedClients[string(routeID)] = append(hr.connectedClients[string(routeID)], ws)
		hr.mu.Unlock()

		// Handle client disconnection
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				hr.mu.Lock()
				hr.removeClient(string(routeID), ws)
				hr.mu.Unlock()
				hr.logger.Info().Str("routeID", string(routeID)).Msg("Client disconnected")
				break
			}
		}
	})

	err := http.ListenAndServe(fmt.Sprintf(":%d", hr.engine.Config.HotReloadServerPort), nil)
	if err != nil {
		hr.logger.Err(err).Msg("Hot reload server quit unexpectedly")
	}
}

// removeClient safely removes a websocket connection from connectedClients
func (hr *HotReload) removeClient(routeID string, ws *websocket.Conn) {
	clients, exists := hr.connectedClients[routeID]
	if !exists {
		return // Route ID not found, nothing to remove
	}

	for i, client := range clients {
		if client == ws {
			hr.connectedClients[routeID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(hr.connectedClients[routeID]) == 0 {
		delete(hr.connectedClients, routeID)
	}
}

// Stop closes all connections and stops the hot reload server
func (hr *HotReload) Stop() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	// Close all client connections
	for _, clients := range hr.connectedClients {
		for _, client := range clients {
			client.Close()
		}
	}

	// Clear the connected clients map
	hr.connectedClients = make(map[string][]*websocket.Conn)
}

// startWatcher starts the file watcher
func (hr *HotReload) startWatcher(basedir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		hr.logger.Err(err).Msg("Failed to start watcher")
		return
	}
	defer watcher.Close()

	// Walk through all files in the frontend directory and add them to the watcher
	err = filepath.Walk(basedir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			hr.logger.Err(err).Msgf("Error accessing path: %s", path)
			return nil
		}
		if fi.Mode().IsDir() {
			if err := watcher.Add(path); err != nil {
				hr.logger.Err(err).Msgf("Failed to add directory to watcher: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		hr.logger.Err(err).Msg("Failed to add files in directory to watcher")
		return
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Check if this is a generated file in .luna directory we should ignore to prevent loops
				normalizedPath := strings.ReplaceAll(event.Name, "\\", "/") // Normalize path separators
				if strings.Contains(normalizedPath, "/.luna/") {
					// Skip rebuild for generated files in .luna directory to prevent infinite loops
					hr.logger.Info().Msgf("Skipping rebuild for .luna file %s", event.Name)
					hr.reloadClients()
					continue
				}

				// Only rebuild if the changed file is a relevant source file
				ext := filepath.Ext(event.Name)
				switch ext {
				case ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".sass", ".less", ".json", ".html":
					hr.logger.Info().Msgf("Detected change in %s, rebuilding...", event.Name)
					// Run InitializeFrontend in a goroutine to avoid blocking the file watcher
					go func() {
						if err := hr.engine.InitializeFrontend(); err != nil {
							hr.logger.Err(err).Msg("Error during frontend initialization")
						}
						hr.reloadClients()
					}()
				case ".go": // Go files changed - may need server restart
					hr.logger.Info().Msgf("Go file changed %s, you may need to restart the server", event.Name)
				default:
					// For other files, just reload without rebuild if they're in the public directory
					hr.logger.Info().Msgf("File changed %s, triggering reload", event.Name)
					hr.reloadClients()
				}
			}
		case err := <-watcher.Errors:
			hr.logger.Err(err).Msg("Error watching files")
		}
	}
}

// reloadClients sends a reload message to all connected clients
func (hr *HotReload) reloadClients() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	for routeID, clients := range hr.connectedClients {
		for i := len(clients) - 1; i >= 0; i-- {
			err := clients[i].WriteMessage(websocket.TextMessage, []byte("reload"))
			if err != nil {
				hr.logger.Err(err).Str("routeID", routeID).Msg("Error sending reload message, removing client")
				hr.removeClient(routeID, clients[i])
			}
		}
	}
}
