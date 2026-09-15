# Release-Plan - DocFlow zur Veröffentlichung

*Stand: 2026-09-15. Nachfolger des ursprünglichen Projektplans (Phase 0-12 alle abgeschlossen).*

## Status

| Aufgabe | Status |
|---------|--------|
| 1. E2E-Validierung am echten Gerät | ✅ erledigt (2026-09-15, live: 2 komplette Workflows Desktop+Handy, Logs personen-frei) |
| 2. CI + Versionierung | ✅ erledigt (2026-09-15, auf privater Gitea: `.gitea/workflows/ci.yml`, CHANGELOG.md, Tag `v1.0.0-rc.1`) |
| 3. Coverage-Lücken schließen | ✅ erledigt (2026-09-15) |
| 4. Handbuch aktualisieren | ✅ erledigt (2026-09-15) |
| 5. Docker-Runtime-Validierung | ⬜ offen (verschoben: kein Docker-Zugang auf dem Arbeitsrechner, `golang:1.21` im Dockerfile muss auf Go 1.26 angehoben werden) |

Nach Aufgabe 1+2 ist das Projekt "veröffentlicht-fähig": getestet,
versioniert, automatisiert geprüft. Offen bleibt nur noch Docker (optional).

## Ausgangslage (Snapshot)

| Bereich | Status |
|---------|--------|
| Features | ✅ alle Phasen + Extras (Tilt-Indikator, Release-Pipeline, i18n, Auto-Config) |
| Tests | ✅ Backend 6/6 Pakete grün, Frontend 119/119, Typecheck 0, Lint 0 |
| Coverage | websocket 94.6%, utils 82.5%, config 78.4%, handlers 78.2%, session 59.4%, cmd/server 55.2% (~Ø 75%) |
| Security | ✅ Constant-time PIN, WS-Auth per Nachricht (kein PIN/IP-Logging), io.LimitReader, Rate-Limiter |
| Privacy | ✅ faktisch verifiziert, zweisprachig (PRIVACY.md + PRIVACY.en.md) |
| Deployment | ✅ Single Binary + Auto-Config + --config/ENV/Flags; Docker strukturell ok (runtime ungetestet) |
| E2E | ⚠️ WS-Auth-Flow nur per Unit-Tests verifiziert, nicht live am Gerät |

---

## Aufgabe 1: E2E-Validierung am echten Gerät — ✅ ERLEDIGT (2026-09-15)

Live durchgeführt (Server lokal, Desktop + Handy im LAN): Session erstellt,
QR gescannt, PIN-Verifizierung, 4 Uploads, Finalize, PDF-Download,
Burn-after-Reading. Kein PIN/Query-String/IP im Access-Log. Dabei wurde
zudem ein Reconnect-Loop-Bug im WS-Client gefunden und behoben
(Commit `16438ac`).

---

## Aufgabe 2: CI + Versionierung — ✅ ERLEDIGT (2026-09-15)

Umgesetzt auf der **privaten Gitea** (`gitea.lan`, Gitea Actions) statt
GitHub - Commit `8051631`:

- `.gitea/workflows/ci.yml`: Backend-Job (npm ci → frontend-build wegen
  `go:embed` → go build, vet, gofmt-Check, Tests mit Coverage) und
  Frontend-Job (npm ci, type-check, lint:check, vitest)
- `CHANGELOG.md` (Keep-a-Changelog): Tag `v1.0.0-rc.1` gesetzt
  (Release-Kandidat; finales v1.0.0 nach Docker-Validierung)

---

## Aufgabe 3: Coverage-Lücken schließen — ✅ ERLEDIGT (2026-09-15)

Umgesetzt in Commits `b0d5c6e` (session) und `c9f89ae` (cmd/server,
runServer-Refactoring). Aktuelle Coverage: session 98.4%, cmd/server 81.9%,
Gesamtprojekt ~85%. Main()-Einstiegspunkt bewusst ungetestet.

## Aufgabe 4: PDF-Handbuch aktualisieren — ✅ ERLEDIGT (2026-09-15)

Umgesetzt in Commit `e9684fc`: Das veraltete Juli-PDF wurde entfernt und
durch `docs/manual.md` als Markdown-Quelle der Wahrheit ersetzt (deckt
Tilt-Indikator, i18n, Auto-Config, WS-Auth, Release-Pipeline ab). Ein
PDF-Export ist optional nachholbar, sobald eine Toolchain verfügbar ist.

## Aufgabe 5: Docker-Runtime-Validierung (~30min, optional) — VERSCHOBEN

Auf dem Arbeitsrechner ist kein Docker-Zugang möglich (User nicht in der
docker-Gruppe). Zusätzlich aufgefallen: Das Dockerfile baut mit
`golang:1.21-alpine`, go.mod verlangt aber Go 1.26.5 - vor der Validierung
anheben (Basis-Image wählen, das Go 1.26 enthält, z.B. golang:1.26-alpine
sobald veröffentlicht, oder aktuelles latest).

---

## Empfohlene Reihenfolge

**5 (Docker)** - der einzige offene Punkt (optional, aufgeschoben bis
Docker auf dem Arbeitsrechner verfügbar ist).

## Regeln (unverändert gültig, siehe AGENTS.md)

- Eine Aufgabe = ein Commit; Tests vor jedem Commit
- Privacy-Grundsätze niemals aufweichen (kein IP/PIN-Logging, RAM-only)
- Dokumentation (README/map/AGENTS/PRIVACY beider Fassungen) im selben
  Commit aktualisieren wie die zugehörige Änderung
