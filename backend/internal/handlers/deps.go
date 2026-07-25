package handlers

import (
	"dokumentenscanner/internal/config"
	"dokumentenscanner/internal/session"
	"dokumentenscanner/internal/websocket"
)

// HandlerDeps enthaelt alle Abhaengigkeiten fur die Handler
type HandlerDeps struct {
	SessionStore *session.Store
	WebSocketHub  *websocket.Hub
	Config        *config.Config
}

// deps ist die globale Instanz der Handler-Deps
// Wird in Init() initialisiert
var deps *HandlerDeps

// Init initialisiert die Handler mit ihren Abhaengigkeiten
// Muss vor dem ersten Handler-Aufruf aufgerufen werden
func Init(store *session.Store, hub *websocket.Hub, cfg *config.Config) {
	deps = &HandlerDeps{
		SessionStore: store,
		WebSocketHub:  hub,
		Config:        cfg,
	}
}

// getDeps gibt die initialisierten Abhaengigkeiten zuruck
// Panics wenn Init() noch nicht aufgerufen wurde
func getDeps() *HandlerDeps {
	if deps == nil {
		panic("handlers.Init() must be called before using handlers")
	}
	return deps
}
