# Dokumentenscanner – Projekt-Dokumentation

*Zentrale Dokumentation für Architektur, Entwicklung und Betrieb. Aktualisiert: 2026-07-25*

---

## Inhaltsverzeichnis

1. [Überblick](#1-überblick)
2. [Architektur](#2-architektur)
3. [Technische Spezifikation](#3-technische-spezifikation)
4. [Konfigurationsmanagement](#4-konfigurationsmanagement)
5. [API-Referenz](#5-api-referenz)
6. [Datenfluss](#6-datenfluss)
7. [Entwicklung](#7-entwicklung)
8. [Deployment](#8-deployment)
9. [Testing](#9-testing)
10. [Bekannte Bugs & Roadmap](#10-bekannte-bugs--roadmap)
11. [Metriken](#11-metriken)
12. [Anhang](#12-anhang)

---

## 1. Überblick

### Projektbeschreibung
Der **Dokumentenscanner** ist eine Web-Anwendung, die den Workflow für das Scannen und Übertragen von Dokumenten zwischen einem **Desktop** (QR-Code-Anzeige) und einem **Mobilgerät** (Kamera + Upload) vereinfacht.

- **Zweck**: Dokumenten-Übertragung ohne direkte Verbindung (z. B. per E-Mail oder Cloud) – ideal für lokale Netzwerke.
- **Zielgruppe**: Nutzer, die schnell Dokumenten-Scans von einem Mobilgerät auf einen Desktop übertragen möchten (z. B. in Meetings oder Homeoffice).

### Kernfeatures
| Feature | Beschreibung |
|---------|--------------|
| **Session-Management** | Temporäre Sessions mit 6-stelliger PIN und 1h Timeout |
| **QR-Code-Generierung** | Automatische Erkennung der LAN-IP für einfache Verbindung |
| **Bild-Upload** | Kamera- oder Datei-Upload mit Client-seitiger Bildbearbeitung (Crop/Rotate) |
| **PDF-Konvertierung** | Server-seitige Generierung eines PDFs aus dem hochgeladenen Bild |
| **Echtzeit-Kommunikation** | WebSocket-basierte Bestätigung zwischen Desktop und Mobile |
| **HTTPS** | Selbstsigniertes TLS-Zertifikat (embedded im Binary) |

---

## 2. Architektur

### Systemarchitektur
```mermaid
graph TD
    subgraph Frontend
        F1[DesktopView]
        F2[MobileView]
    end

    subgraph Backend
        B1["REST API (Gin)"]
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

### Verzeichnisstruktur
```
NeuDocumentenScaner/
├── backend/
│   ├── cmd/server/
│   │   ├── main.go              # Server-Einstiegspunkt (HTTPS, Graceful Shutdown)
│   │   ├── cert.pem             # Self-signed TLS-Zertifikat (embedded)
│   │   ├── key.pem              # TLS-Private Key (embedded)
│   │   ├── index.html           # Frontend SPA (embedded)
│   │   └── assets/              # Frontend-Build-Assets (embedded)
│   │       ├── index-*.js
│   │       └── index-*.css
│   ├── internal/
│   │   ├── config/              # Konfigurationsmanagement (Viper)
│   │   │   ├── config.go        # Config-Struktur + LoadConfig + Defaults
│   │   │   └── config_test.go   # Config-Tests (74% Coverage)
│   │   ├── handlers/            # HTTP-Handler
│   │   │   ├── deps.go          # Dependency Injection
│   │   │   ├── session.go       # CreateSession, VerifyPIN, DeleteSession
│   │   │   ├── qrcode.go        # QR-Code-Generierung (LAN-IP auto-detect)
│   │   │   ├── upload.go        # Bild-Upload + WebSocket-Broadcast
│   │   │   ├── pdf.go           # PDF-Generierung + WebSocket-Broadcast
│   │   │   └── websocket.go     # WebSocket-Handler + Origin-Check + Message-Forwarding
│   │   ├── session/             # Session-Management
│   │   │   ├── session.go       # Session-Struktur + Store (In-Memory)
│   │   │   ├── pin.go           # PIN-Generierung + Verifizierung
│   │   │   ├── cleanup.go       # Session-Timeout (1h) + Auto-Cleanup
│   │   │   ├── session_test.go  # Session-Tests
│   │   │   └── pin_test.go      # PIN-Tests
│   │   └── websocket/           # WebSocket-Hub
│   │       ├── hub.go           # Client-Management + Broadcast
│   │       └── hub_test.go      # Hub-Tests (94% Coverage)
│   ├── pkg/utils/               # Utility-Funktionen
│   │   ├── pdf.go               # PDF-Generierung (gofpdf)
│   │   └── pdf_test.go          # PDF-Tests
│   ├── go.mod                   # Go-Modul
│   ├── go.sum                   # Abhängigkeiten
│   └── Makefile                 # Build-Skripts
├── frontend/
│   └── dokumentenscanner/
│       ├── index.html           # Laedt Inter-Font (Google Fonts)
│       ├── src/
│       │   ├── views/
│       │   │   ├── DesktopView.vue    # QR-Anzeige + Download-Button
│       │   │   └── MobileView.vue     # PIN-Eingabe + Upload + Bestätigung
│       │   ├── components/
│       │   │   ├── QRCodeDisplay.vue  # QR-Code + PIN-Anzeige
│       │   │   ├── PINInput.vue       # 6-stellige PIN-Eingabe mit Lock-Feedback
│       │   │   ├── ImageUpload.vue    # Kamera + Crop + Rotate (Client-seitig)
│       │   │   └── PDFPreview.vue     # PDF-Vorschau
│       │   ├── stores/
│       │   │   └── sessionStore.ts    # Pinia State (Session-ID, Status, Bild)
│       │   ├── utils/
│       │   │   ├── api.ts             # Axios-Client für REST-API
│       │   │   └── websocket.ts       # WebSocket-Client (Message-Handling)
│       │   ├── router/
│       │   │   └── index.ts           # Hash-Routing (Desktop/Mobile)
│       │   ├── assets/
│       │   │   └── main.css           # CSS Custom Properties (Glassmorphism)
│       │   ├── __tests__/             # Frontend-Tests (Vitest)
│       │   │   ├── websocket.test.ts  # WebSocket-Client Tests
│       │   │   ├── sessionStore.test.ts # Pinia Store Tests
│       │   │   ├── api.test.ts        # API-Service Tests
│       │   │   ├── DesktopView.test.ts # Desktop-View Tests
│       │   │   ├── MobileView.test.ts  # Mobile-View Tests
│       │   │   ├── PINInput.test.ts    # PIN-Eingabe Tests
│       │   │   └── QRCodeDisplay.test.ts # QR-Code Tests
│       │   ├── App.vue
│       │   └── main.ts
│       ├── vite.config.ts        # Vite + Vitest Konfiguration
│       └── package.json
└── map.md                        # Diese Datei
```

### Abhängigkeitsgraph (Go-Pakete)
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

---

## 3. Technische Spezifikation

### Backend

#### Framework & Bibliotheken
| Komponente | Technologie | Version | Zweck |
|------------|-------------|---------|-------|
| Web-Framework | Gin | v1.12.0 | HTTP-Routing |
| WebSocket | gorilla/websocket | v1.5.3 | Echtzeit-Kommunikation |
| PDF-Generierung | go-pdf/fpdf | v0.9.0 | PDF-Erstellung aus Bildern |
| QR-Code | skip2/go-qrcode | – | QR-Code-Generierung |
| UUID | google/uuid | – | Session-ID-Generierung |
| Logging | log/slog | stdlib | Strukturiertes Logging |
| TLS | crypto/tls | stdlib | HTTPS mit self-signed Cert |

#### Session-Datenstruktur
```go
type Session struct {
    ID             uuid.UUID
    PIN            string  // 6-stellig, crypto/rand
    Image          []byte  // Hochgeladenes Bild (JPEG/PNG, < 10MB)
    PDF            []byte  // Generiertes PDF
    Status         SessionStatus
    CreatedAt      time.Time
    ExpiresAt      time.Time // 1h nach Erstellung
    FailedAttempts int      // Max. 3 -> 5 Min. Lock
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

- **Speicher**: `map[uuid.UUID]*Session` mit `sync.Mutex` für Thread-Safety (In-Memory).
- **Lebenszyklus**:
  `Erstellung` -> `PIN-Verifizierung` -> `Upload` -> `PDF-Generierung` -> `Download` -> `Automatisches Löschen` (nach 1h oder Download).

#### Sicherheitskonzept
| Massnahme | Implementierung |
|----------|-----------------|
| **PIN-Sicherheit** | 3 Fehlversuche -> 5 Minuten Sperre **pro Session** |
| **Session-Timeout** | 1 Stunde Inaktivitaet **oder** nach PDF-Download |
| **HTTPS** | Selbstsigniertes Zertifikat (im Binary eingebettet) |
| **WebSocket-Auth** | Session-ID im URL-Path (`/ws/session/{id}`) |
| **Bildgroesse** | Max. 10MB pro Upload |

### Frontend

#### Framework & Bibliotheken
| Komponente | Technologie | Version | Zweck |
|------------|-------------|---------|-------|
| Framework | Vue 3 | – | Reaktives UI |
| Build-Tool | Vite | – | Modulares Bundling |
| State-Management | Pinia | – | Zentraler State (Session, Bild) |
| Routing | vue-router | – | Hash-Mode (Desktop/Mobile) |
| HTTP-Client | Axios | – | REST-API-Aufrufe |
| Styling | CSS Custom Properties | – | Glassmorphism-Design |
| Schriftart | Inter | – | Google Fonts |

#### Bildverarbeitung (Client-seitig)
- **APIs**: `ImageBitmap`, `createImageBitmap` (für EXIF-Orientierung)
- **Funktionen**: Kamera-Zugriff, Crop, Rotate

---

## 4. Konfigurationsmanagement

### Architektur

Das Konfigurationssystem nutzt **Viper** mit folgender Prioritätsreihenfolge:

```mermaid
graph TD
    A[CLI Flags] --> B[Umgebungsvariablen]
    B --> C[Config-Datei]
    C --> D[Default-Werte]
    
    style A fill:#e040fb,stroke:#4a148c,color:#fff,stroke-width:2px
    style B fill:#42a5f5,stroke:#0d47a1,color:#fff,stroke-width:2px
    style C fill:#66bb6a,stroke:#1b5e20,color:#fff,stroke-width:2px
    style D fill:#78909c,stroke:#263238,color:#fff,stroke-width:2px
```

### Config-Struktur

```mermaid
graph TD
    root[Config] --> server[Server]
    root --> session[Session]
    root --> upload[Upload]
    root --> websocket[WebSocket]
    root --> logging[Logging]
    
    server --> port["Port: string"]
    server --> host["Host: string"]
    server --> tls_cert_path["TLSCertPath: string"]
    server --> tls_key_path["TLSKeyPath: string"]
    
    session --> timeout["Timeout: duration"]
    session --> cleanup_interval["CleanupInterval: duration"]
    session --> max_failed_attempts["MaxFailedAttempts: int"]
    session --> lockout_duration["LockoutDuration: duration"]
    
    upload --> max_file_size_mb["MaxFileSizeMB: int"]
    upload --> allowed_types["AllowedTypes: string[]"]
    
    websocket --> read_deadline["ReadDeadline: duration"]
    websocket --> ping_interval["PingInterval: duration"]
    websocket --> allowed_origins["AllowedOrigins: string[]"]
    websocket --> allow_private_ips["AllowPrivateIPs: bool"]
    
    logging --> level["Level: string"]
    logging --> format["Format: string"]
```

### Config-Ladevorgang

```mermaid
sequenceDiagram
    participant A as main.go
    participant V as Viper
    participant F as Config-Datei
    participant E as Umgebungsvariablen
    
    A->>V: LoadConfig(configPath)
    V->>V: SetConfigName("config")
    V->>V: SetConfigType("yaml")
    V->>V: AddConfigPath("./config")
    V->>V: AutomaticEnv("DSCAN_")
    V->>F: ReadInConfig()
    alt Datei existiert
        F-->>V: Config-Daten
    else Datei nicht gefunden
        V->>V: Nutze Defaults
    end
    V->>E: bindEnvVars()
    E-->>V: Überschreibe mit ENV
    V->>A: Unmarshal in Config-Struct
    A->>A: validateConfig()
    A-->>A: Return Config
```

### Config-Dateien

| Datei | Umgebungszweck | Besonderheiten |
|-------|---------------|---------------|
| `backend/config/config.yaml` | Standard | Default-Werte für alle Umgebungen |
| `backend/config/config.dev.yaml` | Entwicklung | Debug-Logging, 24h Session-Timeout, 50MB Upload |
| `backend/config/config.prod.yaml` | Produktion | Port 443, TLS-Zertifikate, private IPs gesperrt |

### Docker-Konfiguration

Siehe [Deployment](#8-deployment) für Docker-spezifische Einstellungen.

---

## 5. API-Referenz

### Basis-URL
- **Lokal**: `https://<LAN-IP>:8082` (HTTPS mit self-signed Cert)
- **Beispiel**: `https://192.168.1.100:8082`

### Endpunkte

#### Session-Management
| Methode | Endpunkt | Beschreibung | Request-Body | Response-Body | Status |
|---------|----------|--------------|--------------|----------------|--------|
| POST | `/api/session` | Session erstellen | – | `{"session_id": "uuid", "pin": "123456"}` | 201 |
| DELETE | `/api/session/{id}` | Session loeschen | – | – | 204 |

#### QR-Code & PIN
| Methode | Endpunkt | Beschreibung | Request-Body | Response-Body | Status |
|---------|----------|--------------|--------------|----------------|--------|
| GET | `/api/session/{id}/qrcode` | QR-Code als PNG (enthaelt `session_id` und LAN-IP) | – | PNG-Binary | 200 |
| POST | `/api/session/{id}/verify-pin` | PIN pruefen | `{"pin": "123456"}` | `{"message": "PIN verified"}` | 200 / 403 |

#### Bild-Upload & PDF
| Methode | Endpunkt | Beschreibung | Request-Body | Response-Body | Status |
|---------|----------|--------------|--------------|----------------|--------|
| POST | `/api/session/{id}/upload` | Bild hochladen (JPEG/PNG, < 10MB) | `multipart/form-data` (Field: `image`) | `{"message": "uploaded"}` | 200 / 400 |
| GET | `/api/session/{id}/pdf` | PDF herunterladen | – | PDF-Binary | 200 / 404 |

#### WebSocket
| Methode | Endpunkt | Beschreibung | Nachrichtenformat | Status |
|---------|----------|--------------|-------------------|--------|
| GET | `/ws/session/{id}` | WebSocket-Verbindung fuer Echtzeit-Kommunikation | JSON (siehe unten) | 101 |

**WebSocket-Nachrichten**:
```json
// Server -> Clients (Broadcast)
{"event": "image_uploaded", "session_id": "uuid"}
{"event": "download_request", "session_id": "uuid"}
{"event": "download_confirmed", "session_id": "uuid"}

// Clients -> Server
{"event": "download_request", "data": {"session_id": "uuid"}}
{"event": "download_confirmed", "data": {"session_id": "uuid"}}
```

---

## 6. Datenfluss

### 1. Session-Erstellung + QR-Code
```mermaid
sequenceDiagram
    participant D as Desktop
    participant B as Backend

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

    M->>M: Scan QR -> /#/mobile?session_id=...
    M->>B: POST /api/session/{id}/verify-pin {pin}
    B->>B: Verify PIN
    alt PIN korrekt
        B-->>M: {message: "PIN verified"}
        M->>M: Show Upload UI
        M->>M: Take Photo / Choose File
        M->>B: POST /api/session/{id}/upload (FormData)
        B->>B: Store Image
        B->>D: WebSocket: image_uploaded
        B-->>M: {message: "uploaded"}
        D->>D: Show "Image Received" + Download Button
    else PIN falsch
        B-->>M: {error: "Invalid PIN"} (403)
        M->>M: Show Lock-Feedback (nach 3 Versuchen: 5 Min. Sperre)
    end
```

### 3. Download-Bestaetigung via Mobile
```mermaid
sequenceDiagram
    participant D as Desktop
    participant M as Mobile
    participant B as Backend

    D->>D: User clicks Download
    D->>M: WebSocket: download_request
    M->>M: Show "Download Requested" + Confirm Button
    M->>M: User clicks Confirm
    M->>D: WebSocket: download_confirmed
    D->>B: GET /api/session/{id}/pdf
    B->>B: Generate PDF (gofpdf)
    B-->>D: PDF-Binary
    D->>D: Auto-download PDF via Blob
    D->>B: DELETE /api/session/{id} (optional)
```

### 4. WebSocket Message-Forwarding
```mermaid
sequenceDiagram
    participant C1 as Client 1 (Desktop)
    participant B as Backend (Hub)
    participant C2 as Client 2 (Mobile)

    C1->>B: WebSocket: {"event": "download_request", ...}
    B->>B: Broadcast(sessionID, message)
    B->>C2: WebSocket: {"event": "download_request", ...}
    Note over B: Alle Clients in derselben Session erhalten die Nachricht
```

---

## 7. Entwicklung

### Voraussetzungen
| Tool | Version | Zweck |
|------|---------|-------|
| Go | >= 1.21 | Backend (fuer `log/slog`) |
| Node.js | >= 18 | Frontend (Vue 3 + Vite) |
| npm | >= 9 | Frontend-Dependencies |
| Git | – | Versionskontrolle |

### Projekt-Setup
```bash
# Backend-Dependencies
cd backend
make deps          # Go-Module herunterladen

# Frontend-Dependencies
cd ../frontend/dokumentenscanner
npm install        # Vue/Pinia/Axios/etc.
```

### Build & Run
| Befehl | Beschreibung |
|--------|--------------|
| `make deps` | Go-Module installieren |
| `make lint` | Code-Formatierung pruefen (`go vet` + `gofmt`) |
| `make fmt` | Code automatisch formatieren |
| `make test` | Unit-Tests ausfuehren |
| `make coverage` | Test-Coverage generieren (HTML-Report) |
| `make build` | Backend-Binary erstellen (`./server`) |
| `make run` | Backend starten (HTTPS auf Port 8082) |
| `make frontend-build` | Frontend bauen + nach `cmd/server/` kopieren |
| `make all` | `lint` -> `test` -> `build` |
| `make clean` | Temp-Files, Logs, Binaries bereinigen |

**Manueller Start (ohne Makefile)**:
```bash
# Backend
cd backend
go run ./cmd/server

# Frontend (Dev-Mode mit Hot-Reload)
cd ../frontend/dokumentenscanner
npm run dev
```

### Strukturiertes Logging
- **Bibliothek**: `log/slog` (stdlib, ab Go 1.21)
- **Format**: JSON (maschinenlesbar)
- **Ausgabe**: stdout/stderr (12-Factor-Prinzip, kein File-Logging)

**Log-Levels**:
| Level | Verwendung | Beispiel |
|-------|------------|----------|
| `DEBUG` | WebSocket-Nachrichten, Detail-Infos | `{"level":"DEBUG","msg":"WebSocket message","session_id":"abc-123"}` |
| `INFO` | Server-Start, Session-Erstellung, Upload | `{"level":"INFO","msg":"Session created","session_id":"abc-123"}` |
| `WARN` | Fehlgeschlagene PIN-Versuche, Session-Timeout | `{"level":"WARN","msg":"Invalid PIN attempt","session_id":"abc-123","attempt":2}` |
| `ERROR` | WebSocket-Fehler, PDF-Generierung fehlgeschlagen | `{"level":"ERROR","msg":"PDF generation failed","error":"..."}` |

**Implementiertes Logging:**
- `cmd/server/main.go`: `slog.Info` für Start/Stop
- `handlers/websocket.go`: `slog.Error` für Upgrade-Fehler, `slog.Warn` für Session-Fehler, `slog.Debug` für Nachrichten
- `session/cleanup.go`: `slog.Debug` für Cleanup-Infos

---

## 8. Deployment

### Lokal (Standard)
- **Port**: 8082 (konfigurierbar in `cmd/server/main.go`)
- **HTTPS**: Selbstsigniertes Zertifikat (automatisch generiert bei erstem Start)
- **Start**:
  ```bash
  cd backend
  make run
  ```
- **Zugriff**: `https://localhost:8082` oder `https://<LAN-IP>:8082`

### Reverse-Proxy (optional)
**Beispiel fuer Caddy** (`Caddyfile`):
```
dokumentenscanner.localhost {
    reverse_proxy localhost:8082
}
```
**Beispiel fuer Nginx**:
```nginx
server {
    listen 443 ssl;
    server_name dokumentenscanner.localhost;
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    location / {
        proxy_pass https://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Umgebungsvariablen (geplant)
| Variable | Standardwert | Beschreibung |
|----------|--------------|--------------|
| `PORT` | `8082` | Server-Port |
| `SESSION_TIMEOUT` | `1h` | Session-Lebensdauer |
| `MAX_FAILED_ATTEMPTS` | `3` | PIN-Fehlversuche vor Sperre |
| `LOCKOUT_DURATION` | `5m` | Sperrdauer nach zu vielen Fehlversuchen |

---

## 9. Testing

### Teststruktur
```mermaid
graph TD
    subgraph Backend
        BT[Backend Tests] --> BS[session/pin_test.go]
        BT --> BS2[session/session_test.go]
        BT --> BU[pkg/utils/pdf_test.go]
        BT --> BH[websocket/hub_test.go]
        BT --> BC[config/config_test.go]
        BT --> BI[handlers/integration_test.go]
    end
    subgraph Frontend
        FT[Frontend Tests] --> FW[websocket.test.ts]
        FT --> FS[sessionStore.test.ts]
        FT --> FA[api.test.ts]
        FT --> FD[DesktopView.test.ts]
        FT --> FM[MobileView.test.ts]
        FT --> FP[PINInput.test.ts]
        FT --> FQ[QRCodeDisplay.test.ts]
    end
```

### Test-Coverage

**Backend** (`go test -cover`):

| Paket | Coverage |
|-------|----------|
| `pkg/utils` | 82.4% |
| `internal/config` | 74.5% |
| `internal/session` | 70.0% |
| `internal/handlers` | 68.0% |
| `internal/websocket` | 93.9% |
| `cmd/server` | 14.6% |

**Frontend** (`npm run test:unit`):

| Datei | Tests |
|-------|-------|
| `websocket.test.ts` | 15 |
| `sessionStore.test.ts` | 16 |
| `api.test.ts` | 9 |
| `DesktopView.test.ts` | 9 |
| `MobileView.test.ts` | 8 |
| `PINInput.test.ts` | 12 |
| `QRCodeDisplay.test.ts` | 7 |
| **Gesamt Frontend** | **76** |
| **Gesamt Backend** | **~60** |

### Test ausfuehren

```bash
# Backend
cd backend
go test ./... -v
go test -cover ./...       # Mit Coverage

# Frontend
cd frontend/dokumentenscanner
npx vitest run             # Alle Tests
npx vitest run --reporter=verbose  # Detailliert
```

### Test-Fokus
- **Backend Unit-Tests**: Session-Management, PIN-Generierung, Config-Validierung, WebSocket-Hub
- **Backend Integration-Tests**: Kompletter Workflow (Session -> Upload -> PDF -> Download)
- **Frontend Unit-Tests**: WebSocket-Client, Pinia Store, API-Service
- **Frontend Component-Tests**: DesktopView, MobileView, PINInput, QRCodeDisplay
- **Manuelle Tests**: WebSocket-Kommunikation, Mobile/Desktop-Interaktion im LAN

---

## 10. Bekannte Bugs & Roadmap


### Status der Bugfixes

**Aktualisierung 2026-07-25**: Alle zuvor dokumentierten Bugs wurden behoben + neue fixes ✅

| Bug | Status | Lösung |
|-----|--------|---------|
| Cleanup-Goroutine Leak | ✅ | Session-Store Cleanup in main.go |
| Mutex-Deadlock | ✅ | Redesign der Handler-Dependencies |
| WebSocket Timeouts | ✅ | Config-basierte Timeouts + Ping/Pong |
| WebSocket Origin-Check fehlgeschlagen | ✅ | parseFlags: `ws-allow-private-ips` DefValue-Check ergaenzt (config.go:158) |
| Doppelte WebSocket-Handler-Ausfuehrung | ✅ | Redundanter emitEvent-Aufruf in handleMessage entfernt (websocket.ts) |
| DesktopView Handler-Leak | ✅ | Named Functions + cleanup in onUnmounted (DesktopView.vue) |
| Config-Test fehlgeschlagen | ✅ | Expected AllowedOrigins mit Protokoll-Prefixen aktualisiert |

### Abgeschlossene Phasen

**Phase 3 (Konfigurationsmanagement)** — alle Aufgaben abgeschlossen ✅

| Aufgabe | Beschreibung | Status |
|---------|--------------|--------|
| 3.0 Config-Package | Zentrales Konfigurationsmanagement | ✅ |
| 3.1 Config in main.go | Integration der Config | ✅ |
| 3.2 Handler-Deps | Dependency Injection | ✅ |
| 3.3 Upload-Validierung | Config-basierte Validierung | ✅ |
| 3.4 Origin-Check | Konfigurierbare Origin-Validierung | ✅ |
| 3.5 Config-Dateien | Dev/Prod Config-Templates | ✅ |
| 3.6 Makefile Config | Docker-Build-Ziele | ✅ |
| 3.7 Config-Tests | Unit-Tests für Config | ✅ |
| 3.8 Docker | Containerisierung | ✅ |
| 3.9 Dokumentation | Config-Referenz in Doku | ✅ |

**Phase 4 (Infrastruktur)** — alle Aufgaben abgeschlossen ✅

| Aufgabe | Beschreibung | Status |
|---------|--------------|--------|
| 4.1 Graceful Shutdown | http.Server + SIGINT/SIGTERM + 10s Timeout | ✅ |
| 4.2 Strukturiertes Logging | 0x log.Printf, komplett auf slog umgestellt | ✅ |
| 4.3 Makefile final | lint, fmt, all Targets | ✅ |

**Phase 5 (Testing)** — alle Aufgaben abgeschlossen ✅

| Aufgabe | Beschreibung | Status |
|---------|--------------|--------|
| 5.0 WebSocket-Hub Tests | Register, Unregister, Broadcast (93.9% Coverage) | ✅ |
| 5.1 WebSocket-Handler Tests | isPrivateIP, parseOrigin, createOriginChecker | ✅ |
| 5.2 QRCode-Handler Tests | getFrontendURL, getDefaultPort, getLocalIP | ✅ |
| 5.3 Main/Server Tests | getPort, readEmbeddedFile, setupLogger, writeTempFile | ✅ |
| 5.4 Upload-Handler Tests | 6 Tests: Success, InvalidSession, NotAllowed, InvalidFile, FileExceedsLimit, MissingFormField | ✅ |
| 5.5 Integrationstests | Full Workflow, QR-Code, PIN-Lockout, Concurrent Sessions, WebSocket Broadcast | ✅ |
| 5.6 Config-Tests | bindEnvVars, parseFlags | ✅ |

---

*Letzte Aktualisierung: 2026-07-25. Für den detaillierten Projektplan siehe [plan.md](plan.md).*
