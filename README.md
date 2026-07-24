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

| Variable | Zweck | Default |
|----------|-------|---------|
| `PORT` | Server-Port | `8082` |
| `FRONTEND_URL` | URL im QR-Code | Auto-detect LAN-IP |
| `GIN_MODE` | `release` für Produktion | `debug` |
