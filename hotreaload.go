package luna

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type HotReload struct {
	engine           *Engine
	logger           zerolog.Logger
	connectedClients map[string][]*websocket.Conn
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
		// Client should send routeID as first message
		_, routeID, err := ws.ReadMessage()
		if err != nil {
			hr.logger.Err(err).Msg("Failed to read message from websocket")
			return
		}
		err = ws.WriteMessage(1, []byte("Connected"))
		if err != nil {
			hr.logger.Err(err).Msg("Failed to write message to websocket")
			return
		}
		// Add client to connectedClients
		hr.connectedClients[string(routeID)] = append(hr.connectedClients[string(routeID)], ws)
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", hr.engine.Config.HotReloadServerPort), nil)
	if err != nil {
		hr.logger.Err(err).Msg("Hot reload server quit unexpectedly")
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
	if err = filepath.Walk("frontend/", func(path string, fi os.FileInfo, err error) error {
		if fi.Mode().IsDir() {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		hr.logger.Err(err).Msg("Failed to add files in directory to watcher")
		return
	}

	for {
		select {
		case event := <-watcher.Events:
			// Watch for file created, deleted, updated, or renamed events
			fmt.Println(event.Op)
		case err := <-watcher.Errors:
			hr.logger.Err(err).Msg("Error watching files")
		}
	}
}
