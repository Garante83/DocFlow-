package handlers

import (
	"docflow/internal/config"
	"docflow/internal/session"
	"docflow/internal/websocket"
)

// HandlerDeps holds all dependencies for the handlers
type HandlerDeps struct {
	SessionStore *session.Store
	WebSocketHub *websocket.Hub
	Config       *config.Config
}

// deps is the global instance of the handler deps
// Initialized in Init()
var deps *HandlerDeps

// Init initializes the handlers with their dependencies
// Must be called before the first handler call
func Init(store *session.Store, hub *websocket.Hub, cfg *config.Config) {
	deps = &HandlerDeps{
		SessionStore: store,
		WebSocketHub: hub,
		Config:       cfg,
	}
}

// getDeps returns the initialized dependencies
// Panics if Init() has not been called yet
func getDeps() *HandlerDeps {
	if deps == nil {
		panic("handlers.Init() must be called before using handlers")
	}
	return deps
}
