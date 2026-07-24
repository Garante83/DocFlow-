# Dokumentenscanner – Projektplan

*Atomisierter Plan für devstrahl small. Jeder Block ist ein eigenständiger, umsetzbarer Teilschritt.*

**🔗 Architekturübersicht:** Siehe [map.md](map.md) für Abhängigkeitsgraphen, Datenflussdiagramme und Systemarchitektur.

---

## Phase 0: Technische Spezifikation

### Backend – Framework und Bibliotheken festlegen
**Ziel**: Technologie-Stack für das Backend definieren.

- **Web-Framework**: Gin (`github.com/gin-gonic/gin`)
- **Session-Management**: In-Memory-Store mit Mutex
- **PIN-Generierung**: `crypto/rand` (6-stellig)
- **QR-Code**: `go-qrcode` (`github.com/skip2/go-qrcode`)
- **PDF-Generierung**: `gofpdf` (`github.com/go-pdf/fpdf`)
- **Bildverarbeitung**: Client-seitig via `ImageBitmap` + `createImageBitmap` (EXIF-orientierung)
- **WebSocket**: `gorilla/websocket` (`github.com/gorilla/websocket`)
- **Logging**: Stdlib `log/slog` (ab Go 1.21, kein externes Dependency)

---

### Backend – Session-Datenstruktur definieren
**Ziel**: Datenmodell für Sessions spezifizieren.

```go
type Session struct {
    ID             uuid.UUID
    PIN            string  // 6-stellig
    Image          []byte  // Hochgeladenes Bild
    PDF            []byte  // Generiertes PDF
    Status         SessionStatus
    CreatedAt      time.Time
    ExpiresAt      time.Time // 1h nach Erstellung
    FailedAttempts int
    LockedUntil    time.Time
}

type SessionStatus string

const (
    StatusWaitingForPIN SessionStatus = "waiting_for_pin"
    StatusUploadAllowed SessionStatus = "upload_allowed"
    StatusUploaded      SessionStatus = "uploaded"
    StatusReady         SessionStatus = "ready"
    StatusDownloaded    SessionStatus = "downloaded"
)
```
- **Speicher**: `map[uuid.UUID]*Session` mit `sync.Mutex` für Thread-Safety

---

### Backend – API-Endpunkte spezifizieren
**Ziel**: Alle Backend-Routen und deren Verhalten festlegen.

| Methode | Endpunkt                           | Beschreibung                                    |
|---------|------------------------------------|-------------------------------------------------|
| POST    | `/api/session`                     | Session erstellen → gibt `session_id` + `PIN`    |
| GET     | `/api/session/{id}/qrcode`         | QR-Code als PNG (enthält `session_id`)           |
| POST    | `/api/session/{id}/verify-pin`     | PIN prüfen                                       |
| POST    | `/api/session/{id}/upload`         | Bild hochladen (JPEG/PNG, < 10MB)                |
| GET     | `/api/session/{id}/pdf`            | PDF herunterladen                                |
| DELETE  | `/api/session/{id}`                | Session löschen                                  |
| GET     | `/ws/session/{id}`                 | WebSocket-Verbindung                             |

---

### Backend – Sicherheitskonzept definieren
**Ziel**: Sicherheitsmaßnahmen für Sessions und PINs festlegen.

- **PIN-Sicherheit**: 3 Fehlversuche → 5 Minuten Sperre **pro Session**
- **Session-Timeout**: 1 Stunde Inaktivität **oder** nach PDF-Download
- **HTTPS**: Selbstsigniertes Zertifikat (im Binary eingebettet)
- **WebSocket-Auth**: Session-ID im URL-Path (`/ws/session/{id}`)

---

## Phase 1: Backend-Kernfunktionalität ✅

Alle Aufgaben in Phase 1 sind abgeschlossen:

| Aufgabe | Status |
|---------|--------|
| Backend – Projekt initialisieren | ✅ |
| Backend – Session-Store implementieren | ✅ |
| Backend – PIN-System implementieren | ✅ |
| Backend – Session-Cleanup implementieren | ✅ |
| Backend – Endpunkt Session erstellen | ✅ |
| Backend – Endpunkt QR-Code generieren | ✅ |
| Backend – Endpunkt PIN verifizieren | ✅ |
| Backend – Endpunkt Bild hochladen | ✅ |
| Backend – PDF-Generierung implementieren | ✅ |
| Backend – Endpunkt PDF herunterladen | ✅ |
| Backend – WebSocket-Hub implementieren | ✅ |
| Backend – WebSocket-Endpunkt implementieren | ✅ |
| Backend – Integrationstest Workflow | ✅ |

---

## Phase 2: Frontend-Implementierung ✅

| Aufgabe | Status |
|---------|--------|
| Frontend-Init (Vue 3 + Vite + Pinia) | ✅ |
| DesktopView (QR-Anzeige + Download-Button) | ✅ |
| MobileView (PIN + Upload + Download-Bestätigung) | ✅ |
| QRCodeDisplay (QR-Code + PIN-Anzeige) | ✅ |
| PINInput (6-stellig, Lock-Feedback) | ✅ |
| ImageUpload (Kamera + Crop + Rotate) | ✅ |
| WebSocket-Client (Download-Request/Confirm) | ✅ |
| Moderne UI (Glassmorphism, Inter-Font, SVG-Icons) | ✅ |
| HTTPS (self-signed Zertifikat) | ✅ |

---

## Phase 3: Infrastruktur und Stabilität

### 3.1 Graceful Shutdown
**Ziel**: Sauberes Beenden des Servers ohne Datenverlust.

**Datei**: `cmd/server/main.go`

- `signal.NotifyContext` für SIGINT/SIGTERM
- `http.Server` statt `r.RunTLS()` → `server.Shutdown(ctx)` für in-flight Requests
- Temp-TLS-Files (`cert.pem`, `key.pem`) beim Shutdown aufräumen
- Context an Session-Cleanup weitergeben (Stop-Signal)

**Implementierung**:
```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

srv := &http.Server{Addr: port, Handler: r}
go func() {
    if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
        log.Fatal("Server failed:", err)
    }
}()

<-ctx.Done()
log.Println("Shutting down...")
shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx)
```

**Abhängigkeiten**: Keine

---

### 3.2 Strukturiertes Logging
**Ziel**: Einheitliches, strukturiertes Logging für alle Komponenten.

**Bibliothek**: Stdlib `log/slog` (kein externes Dependency)

**Log-Level**:
| Level | Verwendung |
|-------|------------|
| `DEBUG` | WebSocket-Nachrichten, Detail-Informationen |
| `INFO` | Server-Start, Session-Erstellung, Upload, PDF-Download |
| `WARN` | Fehlgeschlagene PIN-Versuche, Session-Timeout |
| `ERROR` | WebSocket-Fehler, PDF-Generierung fehlgeschlagen |

**Betroffene Dateien**:
| Datei | Aktuelles Logging | Geplantes Logging |
|-------|-------------------|-------------------|
| `cmd/server/main.go` | `log.Printf` (2x) | `slog.Info` für Start, `slog.Error` für Fatal |
| `handlers/session.go` | Keines | `slog.Info` für Create/Delete, `slog.Warn` für PIN-Fehler |
| `handlers/upload.go` | Keines | `slog.Info` für Upload, `slog.Error` für Fehler |
| `handlers/pdf.go` | Keines | `slog.Info` für Download, `slog.Error` für Fehler |
| `handlers/qrcode.go` | Keines | `slog.Info` für QR-Generierung |
| `handlers/websocket.go` | `log.Println` (5x) | `slog.Debug` für Nachrichten, `slog.Error` für Fehler |
| `session/cleanup.go` | `log.Printf` (1x) | `slog.Debug` für Cleanup |
| `websocket/hub.go` | Keines | `slog.Debug` für Register/Unregister |

**Format**: JSON-Struktur für Maschinenlesbarkeit:
```json
{"time":"2026-07-24T12:00:00Z","level":"INFO","msg":"Session created","session_id":"abc-123"}
```

**Kein File-Logging**: Nur stdout/stderr (12-Factor-Prinzip)

**Abhängigkeiten**: Keine

---

### 3.3 Makefile erweitern
**Ziel**: Vollständige Build-Infrastruktur mit Abhängigkeitsverwaltung.

**Datei**: `backend/Makefile`

**Neue Ziele**:

| Ziel | Beschreibung |
|------|-------------|
| `make deps` | Go-Module installieren (`go mod download`) |
| `make lint` | `go vet ./...` + `gofmt -l .` |
| `make fmt` | `gofmt -w .` (automatisches Formatieren) |
| `make frontend-build` | `npm run build` + Copy nach `cmd/server/` |
| `make all` | `lint` → `test` → `build` |
| `make clean` | Erweitert: auch Temp-Files, Logs |

**Vollständige Makefile-Struktur**:
```makefile
.PHONY: deps lint fmt test coverage build run clean all frontend-build

deps:
	go mod download

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:" && gofmt -l . && exit 1)

fmt:
	gofmt -w .

test:
	go test ./... -v

coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

frontend-build:
	cd ../frontend/dokumentenscanner && npm run build
	cp -r ../frontend/dokumentenscanner/dist/* cmd/server/

clean:
	rm -f server coverage.out coverage.html
	rm -f cmd/server.log
	rm -f cmd/cert.pem cmd/key.pem

all: lint test build
```

**Abhängigkeiten**: Keine

---

### 3.4 Bugs fixen
**Ziel**: Bekannte Bugs und Stabilitätsprobleme beheben.

#### Bug 1: Cleanup-Goroutine Leak
**Datei**: `handlers/websocket.go`

**Problem**: `StartCleanup()` wird pro WebSocket-Connection aufgerufen (Zeile 53). Jede Verbindung spawnt einen eigenen, nie-terminierenden Goroutine.

**Fix**: `StartCleanup()` nur noch in `main.go` aufrufen (bereits vorhanden). Zeile 53 in `websocket.go` entfernen.

---

#### Bug 2: Mutex-Deadlock in Hub
**Datei**: `websocket/hub.go`

**Problem**: `broadcastMessage()` (Zeile 80-90) ruft `unregisterClient()` auf, während `h.mu.Lock()` gehalten wird. Go's `sync.Mutex` ist nicht reentrant → Deadlock.

**Fix**: Lock vor Aufruf von `unregisterClient()` freigeben:
```go
func (h *Hub) broadcastMessage(message *Message) {
    h.mu.Lock()
    clients, exists := h.clients[message.sessionID]
    h.mu.Unlock() // ← Lock freigeben

    if !exists {
        return
    }

    for client := range clients {
        if err := client.Conn.WriteMessage(websocket.TextMessage, message.data); err != nil {
            h.Unregister(client) // ← Sendet auf Channel, kein Lock nötig
        }
    }
}
```

---

#### Bug 3: WebSocket Read-Timeout und Keepalive
**Datei**: `handlers/websocket.go`

**Problem**: Keine Read-Deadlines und kein Ping/Pong-Keepalive. Tote TCP-Verbindungen werden nicht erkannt.

**Fix**:
- Read-Deadline setzen (60 Sekunden)
- Ping/Ppong-Handler implementieren
- Bei Pong-Timeout die Verbindung schließen

```go
conn.SetReadDeadline(time.Now().Add(60 * time.Second))
conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    return nil
})

// Ping-Goroutine starten
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
            return
        }
    }
}()
```

**Abhängigkeiten**: Bug 1 (zuerst fixen)

---

## Meilensteine

| Phase | Ziel | Status |
|-------|------|--------|
| Phase 0 | Technische Spezifikation komplett | ✅ |
| Phase 1 | Backend unterstützt kompletten Workflow | ✅ |
| Phase 2 | Frontend implementiert | ✅ |
| Phase 3.1 | Graceful Shutdown | ⬜ |
| Phase 3.2 | Strukturiertes Logging | ⬜ |
| Phase 3.3 | Makefile erweitern | ⬜ |
| Phase 3.4 | Bugs fixen | ⬜ |

---

*Letzte Aktualisierung: 2026-07-24. Siehe `map.md` für Abhängigkeitsvisualisierung.*
