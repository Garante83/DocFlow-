# Dokumentenscanner - Projektplan

*Atomisierter Plan fur Devstral-small. Jede Aufgabe ist ein selbststandiger, umsetzbarer Schritt.*

**Architekturubersicht:** Siehe [map.md](map.md) fur Abhangigkeitsgraphen, Datenflussdiagramme und Systemarchitektur.

---

## Phase 0: Technische Spezifikation

### Backend - Framework und Bibliotheken festlegen
**Ziel**: Technologie-Stack fur das Backend definieren.

- **Web-Framework**: Gin (`github.com/gin-gonic/gin`)
- **Session-Management**: In-Memory-Store mit Mutex
- **PIN-Generierung**: `crypto/rand` (6-stellig)
- **QR-Code**: `go-qrcode` (`github.com/skip2/go-qrcode`)
- **PDF-Generierung**: `gofpdf` (`github.com/go-pdf/fpdf`)
- **Bildverarbeitung**: Client-seitig via `ImageBitmap` + `createImageBitmap` (EXIF-Orientierung)
- **WebSocket**: `gorilla/websocket` (`github.com/gorilla/websocket`)
- **Logging**: Stdlib `log/slog` (ab Go 1.21)
- **Konfiguration**: Viper (`github.com/spf13/viper`)

---

### Backend - Session-Datenstruktur definieren
**Ziel**: Datenmodell fur Sessions spezifizieren.

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

- **Speicher**: `map[uuid.UUID]*Session` mit `sync.Mutex` fur Thread-Safety

---

### Backend - API-Endpunkte spezifizieren
**Ziel**: Alle Backend-Routen und deren Verhalten festlegen.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `/api/session` | Session erstellen -> gibt `session_id` + `PIN` |
| GET | `/api/session/{id}/qrcode` | QR-Code als PNG (enthaelt `session_id`) |
| POST | `/api/session/{id}/verify-pin` | PIN prufen |
| POST | `/api/session/{id}/upload` | Bild hochladen (JPEG/PNG, < 10MB) |
| GET | `/api/session/{id}/pdf` | PDF herunterladen |
| DELETE | `/api/session/{id}` | Session loschen |
| GET | `/ws/session/{id}` | WebSocket-Verbindung |

---

### Backend - Sicherheitskonzept definieren
**Ziel**: Sicherheitsmassnahmen fur Sessions und PINs festlegen.

- **PIN-Sicherheit**: 3 Fehlversuche -> 5 Minuten Sperre **pro Session**
- **Session-Timeout**: 1 Stunde Inaktivitat **oder** nach PDF-Download
- **HTTPS**: Selbstsigniertes Zertifikat (im Binary eingebettet)
- **WebSocket-Auth**: Session-ID im URL-Path (`/ws/session/{id}`)
- **Origin-Check**: Konfigurierbare erlaubte Origins
- **Input-Validierung**: Content-Type und Bildformat-Prufung

---

## Phase 1: Backend-Kernfunktionalitat ✅

Alle Aufgaben in Phase 1 sind abgeschlossen:

| Aufgabe | Status |
|---------|--------|
| Backend - Projekt initialisieren | ✅ |
| Backend - Session-Store implementieren | ✅ |
| Backend - PIN-System implementieren | ✅ |
| Backend - Session-Cleanup implementieren | ✅ |
| Backend - Endpunkt Session erstellen | ✅ |
| Backend - Endpunkt QR-Code generieren | ✅ |
| Backend - Endpunkt PIN verifizieren | ✅ |
| Backend - Endpunkt Bild hochladen | ✅ |
| Backend - PDF-Generierung implementieren | ✅ |
| Backend - Endpunkt PDF herunterladen | ✅ |
| Backend - WebSocket-Hub implementieren | ✅ |
| Backend - WebSocket-Endpunkt implementieren | ✅ |
| Backend - Integrationstest Workflow | ✅ |

---

## Phase 2: Frontend-Implementierung ✅

| Aufgabe | Status |
|---------|--------|
| Frontend-Init (Vue 3 + Vite + Pinia) | ✅ |
| DesktopView (QR-Anzeige + Download-Button) | ✅ |
| MobileView (PIN + Upload + Download-Bestatigung) | ✅ |
| QRCodeDisplay (QR-Code + PIN-Anzeige) | ✅ |
| PINInput (6-stellig, Lock-Feedback) | ✅ |
| ImageUpload (Kamera + Crop + Rotate) | ✅ |
| WebSocket-Client (Download-Request/Confirm) | ✅ |
| Moderne UI (Glassmorphism, Inter-Font, SVG-Icons) | ✅ |
| HTTPS (self-signed Zertifikat) | ✅ |

---

## Phase 3: Produktionsreife & Konfigurationsmanagement ⭐ HOHE PRIORITAT

> **Hinweis fur Devstral-small**: Jede Aufgabe ist atomar und kann einzeln umgesetzt werden.
> **Arbeitsablauf pro Aufgabe**: 1. Aufgabe lesen -> 2. Dateien offnen -> 3. Code implementieren -> 4. Testen -> 5. Commit
>
> **WICHTIG**: Alle in der urprunglichen Planung genannten Bugs (Cleanup-Goroutine Leak, Mutex-Deadlock, WebSocket Timeouts) wurden bereits behoben und sind nicht mehr relevant.

---

### 3.0 Config-Package Grundgerust erstellen
**Ziel**: Zentrales Konfigurationsmanagement fur alle Anwendungseinstellungen.

**Dateien:** `backend/internal/config/config.go` (NEU)

**Anforderungen:**
1. Config Struct mit allen Optionen erstellen
2. `DefaultConfig()` mit Standardwerten implementieren
3. `LoadConfig(configPath string)` implementieren (Datei -> Umgebungsvariablen -> Flags)
4. Validierung hinzufugen

**Akzeptanzkriterien:**
- [x] kompiliert ohne Fehler
- [x] Default-Werte korrekt
- [x] Config aus Datei laden funktioniert

**Abhangigkeiten:** Keine | **Aufwand:** 2-3h

---

### 3.1 Config in main.go integrieren
**Ziel**: Application-Startup mit Konfigurationsmanagement.

**Dateien:** `backend/cmd/server/main.go` (ANPASSEN)

**Anforderungen:**
1. Config am Anfang von main() laden
2. Hardcodierte Werte durch Config-Werte ersetzen
3. Logger basierend auf Config einrichten

**Akzeptanzkriterien:**
- [x] Server startet mit Config-Werten
- [x] Logger nutzt slog mit konfigurierbarem Level

**Abhangigkeiten:** 3.0 | **Aufwand:** 1h

---

### 3.2 Handler-Deps Struct erstellen (Dependency Injection)
**Ziel**: Globale Variablen durch Dependency Injection ersetzen.

**Dateien:** `backend/internal/handlers/*.go` (ANPASSEN)

**Anforderungen:**
1. `HandlerDeps` Struct mit SessionStore, WebSocketHub, Config erstellen
2. Alle Handler nutzen `deps` statt globale Variablen
3. `Init()` Funktion fur Initialisierung

**Akzeptanzkriterien:**
- [x] Alle Handler nutzen deps
- [x] Code kompiliert

**Abhangigkeiten:** 3.0, 3.1 | **Aufwand:** 3-4h

---

### 3.3 Upload-Validierung verbessern
**Ziel**: Konfigurierbare Input-Validierung.

**Dateien:** `backend/internal/handlers/upload.go` (ANPASSEN)

**Anforderungen:**
1. Content-Type gegen Config prufen
2. Dateigroesse gegen Config prufen
3. Klare Fehlermeldungen

**Akzeptanzkriterien:**
- [x] Uploads mit falschem Type abgelehnt
- [x] Uploads uber Maximalgroesse abgelehnt

**Abhangigkeiten:** 3.0, 3.2 | **Aufwand:** 1h

---

### 3.4 WebSocket Origin-Check konfigurierbar machen
**Ziel**: Sichere Origin-Validierung.

**Dateien:** `backend/internal/handlers/websocket.go` (ANPASSEN)

**Anforderungen:**
1. `createOriginChecker(cfg)` Funktion
2. `isPrivateIP(ip)` Hilfsfunktion
3. Upgrader mit CheckOrigin

**Akzeptanzkriterien:**
- [x] Origin-Check konfigurierbar
- [x] Private IPs nur bei Aktivierung

**Abhangigkeiten:** 3.0, 3.2 | **Aufwand:** 1-2h

---

### 3.5 Config-Dateien fur Umgebungen erstellen
**Ziel**: Dev/Prod Config-Dateien.

**Dateien:** `backend/config.*.yaml` (NEU)

**Anforderungen:**
1. `config.default.yaml` mit Standardwerten
2. `config.dev.yaml` fur Entwicklung
3. `config.prod.yaml` fur Produktion

**Akzeptanzkriterien:**
- [x] Alle Dateien sind gultiges YAML
- [x] Configs konnen geladen werden

**Abhangigkeiten:** 3.0 | **Aufwand:** 1h

---

### 3.6 Makefile fur Config erweitern
**Ziel**: Build-Prozess mit Config.

**Dateien:** `backend/Makefile` (ANPASSEN)

**Anforderungen:**
1. Ziele: config-dev, config-prod, build-dev, build-prod

**Akzeptanzkriterien:**
- [x] `make build-dev` funktioniert
- [x] `make build-prod` funktioniert

**Abhangigkeiten:** 3.0, 3.5 | **Aufwand:** 0.5h

---

### 3.7 Config-Tests erstellen
**Ziel**: Unit-Tests fur Config-Package.

**Dateien:** `backend/internal/config/config_test.go` (NEU)

**Anforderungen:**
1. Tests fur DefaultConfig, LoadConfig, Validierung

**Akzeptanzkriterien:**
- [x] Alle Tests passieren

**Abhangigkeiten:** 3.0 | **Aufwand:** 1-2h

---

### 3.8 Docker-Setup aktualisieren
**Ziel**: Container mit Config.

**Dateien:** `Dockerfile`, `docker-compose.yml` (ANPASSEN)

**Anforderungen:**
1. Config als Volume mounten
2. Umgebungsvariablen fur Config

**Akzeptanzkriterien:**
- [x] Container startet mit Config

**Abhangigkeiten:** 3.0, 3.5 | **Aufwand:** 1h

---

### 3.9 Dokumentation aktualisieren
**Ziel**: Config-Referenz.

**Dateien:** `README.md`, `map.md` (ANPASSEN)

**Anforderungen:**
1. Config-Abschnitt in README
2. Config-Diagramm in map.md

**Akzeptanzkriterien:**
- [x] Dokumentation vollständig

**Abhangigkeiten:** 3.0-3.8 | **Aufwand:** 1-2h

---

## Phase 4: Infrastruktur und Stabilitat

---

### 4.2 Strukturiertes Logging implementieren
**Ziel**: Einheitliches Logging.

**Dateien:** Alle Handler und Pakete (ANPASSEN)

**Anforderungen:**
1. Alle `log.Printf` durch `slog.Xxx` ersetzen
2. Sinnvolle Log-Levels nutzen

**Akzeptanzkriterien:**
- [ ] Keine log.Printf mehr
- [ ] Alle Logs strukturiert

**Abhangigkeiten:** 3.1 | **Aufwand:** 2-3h

---

### 4.3 Makefile finalisieren
**Ziel**: Vollstandige Build-Infrastruktur.

**Dateien:** `backend/Makefile` (ANPASSEN)

**Anforderungen:**
1. Alle Ziele prufen und anpassen

**Akzeptanzkriterien:**
- [ ] `make all` funktioniert

**Abhangigkeiten:** 3.6 | **Aufwand:** 0.5h

---

## Status der Bugfixes

**Aktualisierung 2026-07-25**: Alle Bugs behoben:
- Cleanup-Goroutine Leak ✅
- Mutex-Deadlock ✅
- WebSocket Timeouts ✅

---

## Meilensteine

| Phase | Aufgabe | Status | Prioritat | Aufwand | Abhangigkeiten |
|-------|---------|--------|-----------|--------|----------------|
| 0 | Technische Spezifikation | ✅ | - | - | - |
| 1 | Backend-Kernfunktionalitat | ✅ | - | - | - |
| 2 | Frontend-Implementierung | ✅ | - | - | - |
| 3.0 | Config-Package | ✅ | Hoch | 2-3h | - |
| 3.1 | Config in main.go | ✅ | Hoch | 1h | 3.0 |
| 3.2 | Handler-Deps | ✅ | Hoch | 3-4h | 3.0,3.1 |
| 3.3 | Upload-Validierung | ✅ | Hoch | 1h | 3.0,3.2 |
| 3.4 | Origin-Check | ✅ | Hoch | 1-2h | 3.0,3.2 |
| 3.5 | Config-Dateien | ✅ | Hoch | 1h | 3.0 |
| 3.6 | Makefile Config | ✅ | Hoch | 0.5h | 3.0,3.5 |
| 3.7 | Config-Tests | ✅ | Hoch | 1-2h | 3.0 |
| 3.8 | Docker | ✅ | Hoch | 1h | 3.0,3.5 |
| 3.9 | Dokumentation | ✅ | Hoch | 1-2h | 3.0-3.8 |
| 4.1 | Graceful Shutdown | ⬜ | Mittel | 1h | 3.0-3.2 |
| 4.2 | Strukturiertes Logging | ⬜ | Mittel | 2-3h | 3.1 |
| 4.3 | Makefile final | ⬜ | Mittel | 0.5h | 3.6 |

---

## Phase 5: Testverbesserung & Qualitatssicherung

### Uberblick
Aktuelle Testabdeckung: **36.9%** (Ziel: > 80% fur Produktionsreife)
Kritische Lucken in: cmd/server, websocket, handlers/qrcode, handlers/websocket

---

### 5.0 WebSocket-Hub Tests
**Ziel**: 100% Testabdeckung fur WebSocket-Hub-Funktionalitat.

**Dateien:** `backend/internal/websocket/hub_test.go` (NEU)

**Anforderungen:**
1. Test `NewHub()` - Hub Initialisierung
2. Test `Run()` und `Stop()` - Lebenszyklus
3. Test `registerClient()` - Client-Registrierung
4. Test `unregisterClient()` - Client-Entfernung
5. Test `broadcastMessage()` - Nachrichtenverbreitung
6. Test `Unregister()` - Client-Abmeldung

**Akzeptanzkriterien:**
- [ ] Alle Hub-Funktionen > 80% Coverage
- [ ] Parallel testbar (keine Race Conditions)
- [ ] Mock-Clients fur isolierte Tests

**Abhangigkeiten:** Keine | **Aufwand:** 2h

---

### 5.1 WebSocket-Handler Tests
**Ziel**: Testabdeckung fur WebSocket-Endpunkt und Origin-Check.

**Dateien:** `backend/internal/handlers/websocket_test.go` (NEU)

**Anforderungen:**
1. Test `isPrivateIP()` - Private IP-Erkennung
2. Test `createOriginChecker()` - Origin-Check-Funktion
3. Test `parseOrigin()` - URL-Parsing
4. Test `WebSocketHandler()` mit mock Requests
5. Test Origin-Validierung mit verschiedenen Scenarios

**Akzeptanzkriterien:**
- [ ] isPrivateIP: 100% Coverage
- [ ] createOriginChecker: 100% Coverage
- [ ] parseOrigin: 100% Coverage
- [ ] WebSocketHandler: > 80% Coverage

**Abhangigkeiten:** 3.4 | **Aufwand:** 2h

---

### 5.2 Handler QRCode Tests
**Ziel**: Testabdeckung fur QR-Code Handler.

**Dateien:** `backend/internal/handlers/qrcode_test.go` (NEU)

**Anforderungen:**
1. Test `getFrontendURL()` mit/ohne ENV
2. Test `getDefaultPort()` mit/ohne ENV
3. Test `getLocalIP()` - LAN-IP Erkennung
4. Test `QRCodeHandler()` mit validem/invalidem Session-ID

**Akzeptanzkriterien:**
- [ ] Alle qrcode.go Funktionen > 80% Coverage
- [ ] Mock SessionStore fur isolierte Tests

**Abhangigkeiten:** 3.0 | **Aufwand:** 1h

---

### 5.3 Main/Server Tests
**Ziel**: Testabdeckung fur Server-Einstieg und Initialisierung.

**Dateien:** `backend/cmd/server/main_test.go` (NEU)

**Anforderungen:**
1. Test `getPort()` - Port-Extraktion aus Config
2. Test `readEmbeddedFile()` - Dateilesen
3. Test `setupLogger()` - Logger-Konfiguration
4. Test `writeTempFile()` - Temp-Datei-Erstellung
5. Test `main()` Startup-Sequenz (soweit testbar)

**Akzeptanzkriterien:**
- [ ] Alle Helper-Funktionen > 80% Coverage
- [ ] Keine externe Abhangigkeiten in Tests

**Abhangigkeiten:** 3.1 | **Aufwand:** 1h

---

### 5.4 Handler Upload Tests erweiteren
**Ziel**: Erhohte Testabdeckung fur Upload-Handler.

**Dateien:** `backend/internal/handlers/upload_test.go` (NEU)

**Anforderungen:**
1. Test `UploadHandler` mit verschiedenen Dateitypen
2. Test Dateigroessen-Validierung
3. Test Content-Type-Validierung
4. Test Fehlerbehandlung (zu gross, falscher Typ)
5. Test Erfolgsszenario

**Akzeptanzkriterien:**
- [ ] UploadHandler > 90% Coverage
- [ ] Alle Validierungszweige getestet

**Abhangigkeiten:** 3.3 | **Aufwand:** 1h

---

### 5.5 Integrationstests erweiteren
**Ziel**: Vollstandige Workflow-Tests mit Config.

**Dateien:** `backend/internal/handlers/integration_test.go` (ERWEITERN)

**Anforderungen:**
1. Test voller Workflow mit Dev-Config
2. Test voller Workflow mit Prod-Config
3. Test WebSocket-Integration
4. Test Concurrent Sessions

**Akzeptanzkriterien:**
- [ ] Alle Config-Szenarien getestet
- [ ] WebSocket-Integration funktioniert

**Abhangigkeiten:** 3.2, 3.4 | **Aufwand:** 2h

---

### 5.6 Config-Tests erweiteren
**Ziel**: Vollstandige Config-Validierungstests.

**Dateien:** `backend/internal/config/config_test.go` (ERWEITERN)

**Anforderungen:**
1. Test `bindEnvVars()` - Umgebungsvariablen-Binding
2. Test `parseFlags()` - CLI-Flag-Parsing
3. Test Config-Merging (Datei + ENV + Flags)
4. Test Invalid Config-Values

**Akzeptanzkriterien:**
- [ ] Config-Paket > 90% Coverage
- [ ] Alle Validierungsregeln getestet

**Abhangigkeiten:** 3.0 | **Aufwand:** 1h

---

## Meilensteine Testabdeckung

| Phase | Aufgabe | Status | Prioritat | Aufwand | Abhangigkeiten |
|-------|---------|--------|-----------|--------|----------------|
| 5.0 | WebSocket-Hub Tests | ⬜ | Hoch | 2h | Keine |
| 5.1 | WebSocket-Handler Tests | ⬜ | Hoch | 2h | 3.4 |
| 5.2 | Handler QRCode Tests | ⬜ | Hoch | 1h | 3.0 |
| 5.3 | Main/Server Tests | ⬜ | Hoch | 1h | 3.1 |
| 5.4 | Upload Tests erweiteren | ⬜ | Mittel | 1h | 3.3 |
| 5.5 | Integrationstests erweiteren | ⬜ | Mittel | 2h | 3.2, 3.4 |
| 5.6 | Config-Tests erweiteren | ⬜ | Mittel | 1h | 3.0 |

---

## Arbeitsanleitung fur Devstral-small

### Arbeitsablauf:
1. Aufgabe auswählen (nach Abhangigkeiten)
2. Dateien offnen
3. Code implementieren
4. Testen: `go build`, `go test`
5. Commit

### Regeln:
- Eine Aufgabe = Ein Commit
- Immer testen vor naechster Aufgabe
- Keine Abhangigkeiten uberspringen

### Empfohlene Reihenfolge:
3.0 -> 3.1 -> 3.2 -> 3.5 -> 3.3 -> 3.4 -> 3.6 -> 3.7 -> 3.8 -> 3.9 -> 4.1 -> 4.2 -> 4.3 -> 5.0 -> 5.1 -> 5.2 -> 5.3 -> 5.4 -> 5.5 -> 5.6

*Letzte Aktualisierung: 2026-07-25*
### 4.1 Graceful Shutdown finalisieren
**Ziel**: Sauberes Beenden.

**Dateien:** `backend/cmd/server/main.go` (ANPASSEN)

**Anforderungen:**
1. `serveWithGracefulShutdown()` Funktion
2. Signal-Handling fur SIGINT/SIGTERM
3. `http.Server.Shutdown()` nutzen

**Akzeptanzkriterien:**
- [ ] Server endet sauber
- [ ] Offene Verbindungen werden abgeschlossen

**Abhangigkeiten:** 3.0-3.2 | **Aufwand:** 1h

---

### 4.2 Strukturiertes Logging implementieren
**Ziel**: Einheitliches Logging.

**Dateien:** Alle Handler und Pakete (ANPASSEN)

**Anforderungen:**
1. Alle `log.Printf` durch `slog.Xxx` ersetzen
2. Sinnvolle Log-Levels nutzen

**Akzeptanzkriterien:**
- [ ] Keine log.Printf mehr
- [ ] Alle Logs strukturiert

**Abhangigkeiten:** 3.1 | **Aufwand:** 2-3h

---

### 4.3 Makefile finalisieren
**Ziel**: Vollstandige Build-Infrastruktur.

**Dateien:** `backend/Makefile` (ANPASSEN)

**Anforderungen:**
1. Alle Ziele prufen und anpassen

**Akzeptanzkriterien:**
- [ ] `make all` funktioniert

**Abhangigkeiten:** 3.6 | **Aufwand:** 0.5h
