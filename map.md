# Dokumentenscanner - Architekturübersicht

## Systemarchitektur

```mermaid
graph TD
    subgraph Frontend
        F1[DesktopView]
        F2[MobileView]
    end

    subgraph Backend
        B1[REST API<br/>Gin]
        B2[WebSocket Hub]
        B3[Session Store]
        B4[PDF Generator]
        B5[QR Code Generator]
        B6[HTTPS Server]
    end

    F1 -->|HTTP/REST| B1
    F1 -->|WebSocket| B2
    F2 -->|HTTP/REST| B1
    F2 -->|WebSocket| B2
    B1 --> B3
    B1 --> B4
    B1 --> B5
    B2 --> B3
    B6 --> B1
    B6 --> B2
```

## Verzeichnisstruktur (aktuell)

```
NeuDocumentenScaner/
├── backend/
│   ├── cmd/server/
│   │   ├── main.go              # Server-Einstiegspunkt (HTTPS, Graceful Shutdown)
│   │   ├── cert.pem             # Self-signed TLS-Zertifikat (embedded)
│   │   ├── key.pem              # TLS-Private Key (embedded)
│   │   ├── index.html           # Frontend SPA (embedded)
│   │   └── assets/              # Frontend-Build-Assets (embedded)
│   │       ├── index-CicAdPgD.js
│   │       └── index-C0mBOPtv.css
│   ├── internal/
│   │   ├── handlers/            # HTTP-Handler
│   │   │   ├── session.go       # CreateSession, VerifyPIN, DeleteSession
│   │   │   ├── qrcode.go        # QR-Code-Generierung (LAN-IP auto-detect)
│   │   │   ├── upload.go        # Bild-Upload + WebSocket-Broadcast
│   │   │   ├── pdf.go           # PDF-Generierung + WebSocket-Broadcast
│   │   │   └── websocket.go     # WebSocket-Handler + Message-Forwarding
│   │   ├── session/             # Session-Management
│   │   │   ├── session.go       # Session-Struktur + Store (In-Memory)
│   │   │   ├── pin.go           # PIN-Generierung + Verifizierung
│   │   │   └── cleanup.go       # Session-Timeout (1h) + Auto-Cleanup
│   │   └── websocket/           # WebSocket-Hub
│   │       └── hub.go           # Client-Management + Broadcast
│   ├── pkg/utils/               # Utility-Funktionen
│   │   ├── pdf.go               # PDF-Generierung (gofpdf)
│   │   └── pdf_test.go          # PDF-Tests
│   ├── go.mod                   # Go-Modul
│   ├── go.sum                   # Abhängigkeiten
│   └── Makefile                 # Build-Skripts
├── frontend/
│   └── dokumentenscanner/
│       ├── index.html           # Lädt Inter-Font (Google Fonts)
│       ├── src/
│       │   ├── views/
│       │   │   ├── DesktopView.vue    # QR-Anzeige + Download
│       │   │   └── MobileView.vue     # PIN + Upload + Confirm
│       │   ├── components/
│       │   │   ├── QRCodeDisplay.vue  # QR-Code + PIN-Anzeige
│       │   │   ├── PINInput.vue       # 6-stellige PIN-Eingabe
│       │   │   └── ImageUpload.vue    # Kamera + Crop + Rotate
│       │   ├── stores/
│       │   │   └── sessionStore.ts    # Pinia State
│       │   ├── utils/
│       │   │   ├── api.ts             # Axios-Client
│       │   │   └── websocket.ts       # WebSocket-Client
│       │   ├── router/
│       │   │   └── index.ts           # Hash-Routing (Desktop/Mobile)
│       │   ├── assets/
│       │   │   └── main.css           # CSS Custom Properties
│       │   ├── App.vue
│       │   └── main.ts
│       └── vite.config.ts
└── map.md                        # Diese Datei
```

## Abhängigkeitsgraph (Go-Pakete)

```mermaid
graph TD
    M[cmd/server/main.go] --> H[internal/handlers]
    M --> S[internal/session]
    M --> W[internal/websocket]
    H --> S
    H --> W
    H --> U[pkg/utils]
    W --> WS[gorilla/websocket]
    S --> UUID[google/uuid]
    U --> PDF[go-pdf/fpdf]
    H --> QR[skip2/go-qrcode]
    H --> GIN[gin-gonic/gin]
```

## Paketabhängigkeiten (Detail)

### cmd/server/main.go
- **Zweck**: Server-Eintrittspunkt, HTTPS, Embedding
- **Abhängigkeiten**:
  - `internal/handlers` → HTTP-Handler
  - `internal/session` → Session-Store
  - `internal/websocket` → WebSocket-Hub
  - `github.com/gin-gonic/gin` → Web-Framework

### internal/handlers/*.go
- **Zweck**: HTTP-Request-Handler
- **Abhängigkeiten**:
  - `internal/session` → Session-Management
  - `pkg/utils` → PDF-Generierung
  - `github.com/gin-gonic/gin` → Routing
  - `github.com/gorilla/websocket` → WebSocket
  - `github.com/skip2/go-qrcode` → QR-Code

### internal/session/*.go
- **Zweck**: Session-Lebenszyklus-Management
- **Abhängigkeiten**:
  - `github.com/google/uuid` → Session-IDs
  - `time` → Timeouts
  - `sync` → Thread-Safety

### internal/websocket/hub.go
- **Zweck**: Echtzeit-Kommunikation
- **Abhängigkeiten**:
  - `github.com/gorilla/websocket` → WebSocket-Implementierung

### pkg/utils/*.go
- **Zweck**: PDF-Generierung
- **Abhängigkeiten**:
  - `github.com/go-pdf/fpdf` → PDF-Generierung

## Datenfluss

### 1. Session-Erstellung + QR-Code
```mermaid
sequenceDiagram
    participant D as Desktop
    participant B as Backend
    participant M as Mobile

    D->>B: POST /api/session
    B->>B: Create Session + PIN
    B-->>D: {session_id, pin}
    D->>B: GET /api/session/{id}/qrcode
    B->>B: Generate QR (https://LAN:8082/#/mobile?session_id=...)
    B-->>D: QR-Code PNG
    D->>D: Display QR + PIN
```

### 2. Mobile: PIN + Upload
```mermaid
sequenceDiagram
    participant M as Mobile
    participant B as Backend
    participant D as Desktop

    M->>M: Scan QR → /#/mobile?session_id=...
    M->>B: POST /api/session/{id}/verify-pin {pin}
    B->>B: Verify PIN
    B-->>M: {message: "PIN verified"}
    M->>M: Show Upload UI
    M->>M: Take Photo / Choose File
    M->>B: POST /api/session/{id}/upload (FormData)
    B->>B: Store Image
    B->>D: WebSocket: image_uploaded
    B-->>M: {message: "uploaded"}
    D->>D: Show "Image Received" + Download Button
```

### 3. Download-Bestätigung via Mobile
```mermaid
sequenceDiagram
    participant D as Desktop
    participant M as Mobile
    participant B as Backend

    D->>D: User clicks Download
    D->>M: WebSocket: download_request
    M->>M: Show "Download Requested" + Confirm Button
    M->>B: GET /api/session/{id}/pdf
    B->>B: Generate PDF
    B-->>M: PDF-File
    M->>M: User clicks Confirm
    M->>D: WebSocket: download_confirmed
    D->>D: Auto-download PDF via Blob
```

### 4. WebSocket Message-Forwarding
```mermaid
sequenceDiagram
    participant C1 as Client 1
    participant B as Backend (Hub)
    participant C2 as Client 2

    C1->>B: WebSocket: {"type": "...", ...}
    B->>B: Broadcast(sessionID, message)
    B->>C2: WebSocket: {"type": "...", ...}
    Note over B: Alle Clients in derselben<br/>Session erhalten die Nachricht
```

## Teststruktur

```mermaid
graph TD
    A[Unit Tests] --> B[internal/session/pin_test.go]
    A --> C[internal/session/session_test.go]
    A --> D[pkg/utils/pdf_test.go]
    E[Integration Tests] --> F[internal/handlers/integration_test.go]
```

## Metriken (aktuell)

| Metrik | Wert |
|--------|------|
| Go-Dateien | 14 |
| Frontend-Dateien | 12 (Vue/TS) |
| Testdateien | 4 |
| Unit-Tests | 11 |
| Integration-Tests | 1 |
| Externe Dependencies | 6 (Go) |

## Technologie-Stack

| Komponente | Technologie |
|------------|-------------|
| Backend-Framework | Gin v1.12.0 |
| Frontend-Framework | Vue 3 + Vite |
| State-Management | Pinia |
| Routing | vue-router (Hash-Mode) |
| WebSocket | gorilla/websocket v1.5.3 |
| PDF-Generierung | go-pdf/fpdf v0.9.0 |
| QR-Code | skip2/go-qrcode |
| TLS | Self-signed (im Binary embedded) |
| Logging | log/slog (stdlib) |
| Fonts | Inter (Google Fonts) |

## Bekannte Bugs (Phase 3.4)

| Bug | Datei | Beschreibung |
|-----|-------|-------------|
| Cleanup-Goroutine Leak | handlers/websocket.go | `StartCleanup()` wird pro WS-Connection aufgerufen |
| Mutex-Deadlock | websocket/hub.go | `broadcastMessage()` ruft `unregisterClient()` mit aktivem Lock auf |
| WebSocket Timeouts | handlers/websocket.go | Keine Read-Deadlines, kein Ping/Pong |

---

*Letzte Aktualisierung: 2026-07-24*
