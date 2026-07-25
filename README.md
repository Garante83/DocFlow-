# NDscaner – Dokumentenscanner

Web-basierter Dokumentenscanner: Desktop zeigt QR-Code, Handy scannt und lädt Bilder hoch, Desktop generiert PDF.

## Stack

- **Backend**: Go + Gin, WebSocket, PDF-Generierung (gofpdf)
- **Frontend**: Vue 3 + Vite + Pinia, Hash-Routing
- **TLS**: Selbstsigniertes Zertifikat (im Binary eingebettet)

## Quick Start

```bash
cd backend

# Dependencies
make deps

# Build
make all

# Run
make run
```

Server: `https://localhost:8082`

## Build & Deploy

```bash
cd backend

# Frontend bauen + einbetten
make frontend-build

# Alles kompilieren
make all

# Server starten
./server
```

## API

| Methode | Endpoint | Beschreibung |
|---------|----------|-------------|
| POST | `/api/session` | Session erstellen |
| GET | `/api/session/{id}/qrcode` | QR-Code als PNG |
| POST | `/api/session/{id}/verify-pin` | PIN verifizieren |
| POST | `/api/session/{id}/upload` | Bild hochladen |
| GET | `/api/session/{id}/pdf` | PDF herunterladen |
| DELETE | `/api/session/{id}` | Session löschen |
| GET | `/ws/session/{id}` | WebSocket-Verbindung |

## Konfiguration

Der Dokumentenscanner verwendet **Viper** für zentrales Konfigurationsmanagement mit den folgenden Quellen (in Prioritätsreihenfolge):
1. **CLI-Flags** (z.B. `--port`, `--host`)
2. **Umgebungsvariablen** (Praefix: `DSCAN_`)
3. **Config-Dateien** (YAML)
4. **Default-Werte** (hardcodet)

---

### Umgebungsvariablen

Alle Konfigurationsoptionen können über Umgebungsvariablen mit dem Praefix `DSCAN_` gesetzt werden:

| Umgebungsvariable | Config-Pfad | Standardwert | Beschreibung |
|-------------------|-------------|--------------|--------------|
| `DSCAN_SERVER_PORT` | `server.port` | `8082` | Server-Port |
| `DSCAN_SERVER_HOST` | `server.host` | `0.0.0.0` | Server-Host |
| `DSCAN_SERVER_TLS_CERT_PATH` | `server.tls_cert_path` | `` | Pfad zum TLS-Zertifikat |
| `DSCAN_SERVER_TLS_KEY_PATH` | `server.tls_key_path` | `` | Pfad zum TLS-Private-Key |
| `DSCAN_SESSION_TIMEOUT` | `session.timeout` | `1h` | Session-Lebensdauer |
| `DSCAN_SESSION_CLEANUP_INTERVAL` | `session.cleanup_interval` | `5m` | Intervall für Session-Bereinigung |
| `DSCAN_SESSION_MAX_FAILED_ATTEMPTS` | `session.max_failed_attempts` | `3` | Max. PIN-Fehlversuche |
| `DSCAN_SESSION_LOCKOUT_DURATION` | `session.lockout_duration` | `5m` | Sperrdauer nach Lockout |
| `DSCAN_UPLOAD_MAX_FILE_SIZE_MB` | `upload.max_file_size_mb` | `10` | Max. Dateigröße in MB |
| `DSCAN_UPLOAD_ALLOWED_TYPES` | `upload.allowed_types` | `image/jpeg,image/png,image/webp` | Erlaubte MIME-Typen (kommagetrennt) |
| `DSCAN_WEB_SOCKET_READ_DEADLINE` | `websocket.read_deadline` | `60s` | WebSocket Read-Timeout |
| `DSCAN_WEB_SOCKET_PING_INTERVAL` | `websocket.ping_interval` | `30s` | WebSocket Ping-Intervall |
| `DSCAN_WEB_SOCKET_ALLOWED_ORIGINS` | `websocket.allowed_origins` | `localhost:8082,127.0.0.1:8082` | Erlaubte Origins (kommagetrennt) |
| `DSCAN_WEB_SOCKET_ALLOW_PRIVATE_IPS` | `websocket.allow_private_ips` | `true` | Private IPs erlauben |
| `DSCAN_LOGGING_LEVEL` | `logging.level` | `info` | Log-Level (debug, info, warn, error) |
| `DSCAN_LOGGING_FORMAT` | `logging.format` | `json` | Log-Format (json, text) |

---

### Config-Dateien

Vordefinierte Config-Dateien im Verzeichnis `backend/config/`:

| Datei | Verwendung | Beschreibung |
|-------|-----------|--------------|
| `config.yaml` | Standard | Default-Werte für alle Umgebungen |
| `config.dev.yaml` | Entwicklung | Debug-Logging, längere Timeouts, größere Upload-Limits |
| `config.prod.yaml` | Produktion | HTTPS auf Port 443, striktere Sicherheitseinstellungen |

**Config-Datei laden:**
```bash
# Mit Dev-Config starten
cp backend/config/config.dev.yaml backend/config/config.yaml
cd backend && ./server

# Oder via Docker mit gemountetem Volume
make docker-run  # Lädt automatisch config.yaml
```

**Beispiel Config-Datei (config.yaml):**
```yaml
server:
  port: "8082"
  host: "0.0.0.0"
  tls_cert_path: ""
  tls_key_path: ""

session:
  timeout: 1h
  cleanup_interval: 5m
  max_failed_attempts: 3
  lockout_duration: 5m

upload:
  max_file_size_mb: 10
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "image/webp"

websocket:
  read_deadline: 60s
  ping_interval: 30s
  allowed_origins:
    - "localhost:8082"
    - "127.0.0.1:8082"
  allow_private_ips: true

logging:
  level: "info"
  format: "json"
```

---

### CLI-Flags

| Flag | Beschreibung | Standardwert |
|------|--------------|--------------|
| `--port` | Server-Port | `8082` |
| `--host` | Server-Host | `0.0.0.0` |
| `--config` | Pfad zur Config-Datei | `` |
| `--ws-allow-private-ips` | Private IPs für WebSocket erlauben | `true` |
| `--ws-origins` | Erlaubte Origins (kommagetrennt) | `localhost:8082,127.0.0.1:8082` |
