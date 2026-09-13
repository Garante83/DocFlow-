# Release-Plan - DocFlow zur Veröffentlichung

*Stand: 2026-09-13. Nachfolger des ursprünglichen Projektplans (Phase 0-12 alle abgeschlossen).*

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

## Aufgabe 3: Coverage-Lücken schließen (~1.5h)

Fokus auf die zwei schwächsten Pakete:

- `internal/session` (59.4%): Tests für Cleanup-Goroutine (expired/downloaded/
  active), TOCTOU-`UpdateFunc`, PIN-Hashing-Grenzfälle
- `cmd/server` (55.2%): Helper-Tests (setupLogger-Formate, preScanConfigPath
  Varianten - teils vorhanden), ggf. `runServer`-Einstieg über httptest

**Akzeptanzkriterien:** beide Pakete > 75%, Gesamtprojekt > 80%.

---

## Aufgabe 4: PDF-Handbuch aktualisieren (~2h, optional)

`docs/Dokumentenscanner-Dokumentation.pdf` stammt vom Juli-Stand und kennt
nicht: Tilt-Indikator, Mehrsprachigkeit, Auto-Config/`--config`, neue
WS-Auth, Release-Pipeline.

Quelle der Wahrheit: `map.md` (Architektur), `README.md` (Betrieb),
`AGENTS.md` (Kommandos). PDF neu generieren oder Handbuch als Markdown in
`docs/` pflegen und PDF nur als Export.

**Akzeptanzkriterien:** Handbuch deckt alle Nutzer-Features ab; Datum aktualisiert.

---

## Aufgabe 5: Docker-Runtime-Validierung (~30min, optional)

Dockerfile ist Multi-Stage (node → golang → alpine), aber nie live gelaufen:

1. `docker build -t docflow .`
2. `docker run` → HTTPS auf 8082 erreichbar, Auto-Config funktioniert im
   Container (read-only-FS-Fallback prüfen)
3. `docker-compose up` → Volume-Mount `./backend/config` verhält sich korrekt

**Akzeptanzkriterien:** Image startet; Workflow läuft im Container.

---

## Empfohlene Reihenfolge

**1 (E2E) → 2 (CI/Tag) → 3 (Coverage) → 5 (Docker) → 4 (Handbuch)**

Gesamt: ~6h. Nach 1+2 ist das Projekt offiziell "veröffentlicht-fähig":
getestet, versioniert, automatisiert geprüft.

## Regeln (unverändert gültig, siehe AGENTS.md)

- Eine Aufgabe = ein Commit; Tests vor jedem Commit
- Privacy-Grundsätze niemals aufweichen (kein IP/PIN-Logging, RAM-only)
- Dokumentation (README/map/AGENTS/PRIVACY beider Fassungen) im selben
  Commit aktualisieren wie die zugehörige Änderung
