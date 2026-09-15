# Release-Plan - DocFlow zur Veröffentlichung

*Stand: 2026-09-15. Nachfolger des ursprünglichen Projektplans (Phase 0-12 alle abgeschlossen).*

## Status

| Aufgabe | Status |
|---------|--------|
| 1. E2E-Validierung am echten Gerät | ⬜ offen (höchste Priorität) |
| 2. CI + Versionierung | ⬜ offen |
| 3. Coverage-Lücken schließen | ✅ erledigt (2026-09-15) |
| 4. Handbuch aktualisieren | ✅ erledigt (2026-09-15) |
| 5. Docker-Runtime-Validierung | ⬜ offen (verschoben: kein Docker-Zugang auf dem Arbeitsrechner, `golang:1.21` im Dockerfile muss auf Go 1.26 angehoben werden) |

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

## Aufgabe 1: E2E-Validierung am echten Gerät (~30 min) — HÖCHSTE PRIORITÄT

Der WebSocket-Auth-Wechsel (PIN als erste Nachricht statt Query-Parameter) ist
das größte Design-Änderung des letzten Durchgangs und muss live durchgespielt
werden:

1. `make release`, Server starten (LAN)
2. Desktop: Session erstellen, QR-Code erscheint, **kein** PIN im
   Browser-Network-Tab bei der WS-Verbindung (Pfad ohne `?token=`)
3. Handy: QR scannen, PIN-Eingabe, Kamera + Tilt-Indikator (Sensor oder
   Bildanalyse-Fallback je Browser), Upload mehrerer Seiten
4. Desktop: Seitenzähler aktualisiert sich (WS-Broadcast funktioniert nach Auth)
5. PDF generieren → Download → Session gelöscht (Burn-after-Reading bestätigen)
6. Log-Check: keine IP, kein `token=`, kein PIN in der Ausgabe

**Akzeptanzkriterien:** kompletter Workflow läuft; Logs bleiben personen-frei.

---

## Aufgabe 2: CI + Versionierung (~1.5h)

**2a. GitHub Actions** (`.github/workflows/ci.yml`):

```yaml
Backend-Job:  setup-go (1.26), cd backend, go build, go vet, gofmt -l check, go test ./... -coverprofile
Frontend-Job: setup-node (22), npm ci, npm run type-check, npm run lint:check, npm run test:unit -- --run
```

Hinweis: `go build`/`go test` im CI brauchen zuerst `make frontend-build`
(Node >= 22) - Reihenfolge im Workflow beachten (siehe AGENTS.md).

**2b. Git-Tag + CHANGELOG:**
- `CHANGELOG.md` anlegen (Keep-a-Changelog-Format): Highlight-Einträge
  i18n, Tilt-Indikator, Single-Binary-Release, Config-System, Privacy-Logging
- Tag `v1.0.0` setzen (Release-Kandidat) oder `v0.1.0` wenn erst Beta

**2c. Release-Artefakte:** `make release-linux` Binary als GitHub-Release
anhängen (optional, ~15min)

**Akzeptanzkriterien:** CI grün auf push; Tag existiert; Changelog dokumentiert.

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

**1 (E2E) → 2 (CI/Tag) → 5 (Docker, verschoben)**

Nach 1+2 ist das Projekt offiziell "veröffentlicht-fähig":
getestet, versioniert, automatisiert geprüft.

## Regeln (unverändert gültig, siehe AGENTS.md)

- Eine Aufgabe = ein Commit; Tests vor jedem Commit
- Privacy-Grundsätze niemals aufweichen (kein IP/PIN-Logging, RAM-only)
- Dokumentation (README/map/AGENTS/PRIVACY beider Fassungen) im selben
  Commit aktualisieren wie die zugehörige Änderung
