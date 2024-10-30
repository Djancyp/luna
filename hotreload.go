package luna

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
func (hr *HotReload) Start() {
	go hr.startServer()
	go hr.startWatcher()
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
	clients := hr.connectedClients[routeID]
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

// startWatcher starts the file watcher
func (hr *HotReload) startWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		hr.logger.Err(err).Msg("Failed to start watcher")
		return
	}
	defer watcher.Close()

	// Walk through all files in the frontend directory and add them to the watcher
	err = filepath.Walk("frontend/", func(path string, fi os.FileInfo, err error) error {
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
				hr.logger.Info().Msg("Detected change, reloading clients...")
				hr.engine.InitializeFrontend()
				hr.reloadClients()
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
