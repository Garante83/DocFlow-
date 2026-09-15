# DocFlow - Benutzerhandbuch

*Stand: 2026-09-15. Dieses Handbuch ist die Quell-Dokumentation für Nutzer
und Betreiber. Architektur-Details: [map.md](../map.md), Konfiguration im
Detail: [README.md](../README.md) "Configuration", Build-Kommandos:
[AGENTS.md](../AGENTS.md).*

---

## 1. Was ist DocFlow?

DocFlow ist ein web-basierter Dokumentenscanner: Der Desktop zeigt einen
QR-Code, das Handy verbindet sich über das lokale Netzwerk, macht Fotos und
lädt sie hoch - der Server erzeugt daraus ein mehrseitiges PDF. Alles ohne
Cloud, ohne E-Mail, ohne Installation auf dem Handy.

| Feature | Beschreibung |
|---------|--------------|
| Session-Management | Temporäre Sessions mit 6-stelliger PIN und konfigurierbarem Timeout |
| QR-Code-Verbindung | Automatische LAN-IP-Erkennung, Handy verbindet sich per Scan |
| Multi-Page-Upload | Mehrere Fotos pro Session, bearbeitbar (drehen, zuschneiden) |
| Kipp-Indikator | Live-Hinweis beim Fotografieren, ob das Dokument gerade im Rahmen liegt |
| Mehrsprachigkeit | Deutsch und Englisch, automatisch je Browser-Sprache |
| PDF-Erzeugung | Mehrseitiges PDF mit JPEG-Komprimierung (Standard 85%) |
| Echtzeit | WebSocket-Updates: Desktop sieht sofort, wenn eine Seite ankommt |
| Burn-after-Reading | Session inkl. aller Bilder wird nach dem PDF-Download gelöscht |
| HTTPS | Selbstsigniertes Zertifikat wird beim Start automatisch erzeugt |

---

## 2. Kurzanleitung: Der Workflow

### Desktop

1. **Server starten** (siehe Kapitel 3) und
   `https://<host>:8082` im Browser öffnen - die
   Selbstsigniert-Zertifikat-Warnung einmalig bestätigen
2. **Session erstellen** - QR-Code und PIN erscheinen
3. Warten, bis das Handy Fotos hochlädt (Seitenzähler aktualisiert sich
   live)
4. Nach dem "Fertigstellen" auf dem Handy: **PDF herunterladen**
5. Die Session wird automatisch gelöscht (Burn-after-Reading)

### Handy

1. **QR-Code scannen** mit der Kamera-App - der Browser öffnet die
   Mobile-Anzeige
2. **6-stellige PIN eingeben** (steht neben dem QR-Code auf dem Desktop) -
   nach drei Fehlversuchen wird die Eingabe vorübergehend gesperrt
3. **Fotos machen** oder aus der Galerie wählen - mehrere Seiten sind
   möglich, jede Seite kann vor dem Hochladen gedreht oder zugeschnitten
   werden
4. **"Fertig"** tippen - der Server erzeugt das PDF
5. Sobald der Desktop den Download anfordert, wird das bestätigt; die
   Mobile-Anzeige zeigt "Vorgang abgeschlossen"

### Kipp-Indikator beim Fotografieren

Während die Kamera offen ist, zeigt DocFlow live an, ob das Dokument gerade
liegt und den Rahmen ausfüllt (Status gut/ok/schlecht + "gefüllt"). Die
Erkennung arbeitet dreistufig mit automatischem Fallback:

1. Gerätesensor (`deviceorientation`) - iOS mit erteilter Erlaubnis, Firefox
2. Accelerometer (Generic Sensor) - Chrome/Android
3. Bildanalyse - immer verfügbar, auch wenn ein Browser Sensoren still
   blockiert (z.B. Brave); funktioniert über Grauwert-Segmentierung und
   Rechteck-/Perspektiven-Prüfung direkt im Browser

Alle drei Wege laufen komplett clientseitig - die Bilder werden dafür nicht
hochgeladen.

### Sprache

Die Oberfläche ist auf Deutsch und Englisch verfügbar; die Sprache wird
automatisch aus der Browsereinstellung gewählt.

---

## 3. Installation & Start

### Variante A: Single Binary (empfohlen)

```bash
# Selbst bauen (benötigt Go >= 1.26 und Node >= 22)
make release          # aus dem Projektstamm

# Oder cross-kompilieren
make release-linux    # -> backend/docflow-linux-amd64
```

Start:

```bash
./docflow
# → https://localhost:8082
```

Beim ersten Start ohne Konfigurationsdatei wird automatisch eine komplett
kommentierte `config.yaml` an den ersten schreibbaren Ort geschrieben
(`./`, `./config/`, `/etc/docflow/`). Eine bestehende Datei wird nie
überschrieben; auf schreibgeschützten Dateisystemen läuft der Server mit
integrierten Standardwerten weiter.

### Variante B: Entwicklung

```bash
cd frontend/dokumentenscanner
npm install
npm run dev           # Dev-Server mit Hot-Reload

cd backend
make frontend-build   # Frontend bauen und einbetten (einmalig nötig)
make run              # HTTPS auf Port 8082
```

### Variante C: Docker

```bash
docker build -t docflow .
docker run -p 8082:8082 docflow
```

Das Image ist Multi-Stage (Node → Go → Alpine) und läuft mit einem
nicht-privilegierten Benutzer. Hinweis: Die Docker-Variante ist strukturell
fertig, wurde aber noch nicht am laufenden System validiert (siehe
`docs/release-plan.md`, Aufgabe 5).

---

## 4. Konfiguration

Alle Einstellungen lassen sich über vier Quellen setzen, Priorität von oben
nach unten:

1. **CLI-Flags**: `--port`, `--host`, `--config <pfad>`,
   `--ws-origins`, `--ws-allow-private-ips`
2. **Umgebungsvariablen** mit Präfix `DSCAN_`
   (z.B. `DSCAN_SERVER_PORT=9000`)
3. **YAML-Datei** (Suchpfade: `./`, `./config/`, `/etc/docflow/`, oder
   explizit per `--config`)
4. **Eingebaute Standardwerte**

Die wichtigsten Einstellungen (Auszug; vollständig kommentiert in der
auto-generierten `config.yaml` und in README "Configuration"):

| Einstellung | ENV-Variable | Standard |
|-------------|--------------|----------|
| Port | `DSCAN_SERVER_PORT` | `8082` |
| Session-Timeout | `DSCAN_SESSION_TIMEOUT` | `1h` |
| PIN-Fehlversuche bis Sperre | `DSCAN_SESSION_MAX_FAILED_ATTEMPTS` | `3` |
| Sperrdauer | `DSCAN_SESSION_LOCKOUT_DURATION` | `5m` |
| Max. Dateigröße pro Bild | `DSCAN_UPLOAD_MAX_FILE_SIZE_MB` | `10` |
| Max. Seiten pro PDF | `DSCAN_PDF_MAX_PAGES` | `20` |
| JPEG-Qualität | `DSCAN_PDF_JPEG_QUALITY` | `85` |
| Zulässige WS-Origins | `DSCAN_WEB_SOCKET_ALLOWED_ORIGINS` | localhost |
| Log-Level / -Format | `DSCAN_LOGGING_LEVEL` / `_FORMAT` | `info` / `json` |
| Rate-Limiting | `DSCAN_RATE_LIMIT_*` | aktiv, 100 Req./60s |

Vorgefertigte Profile liegen in `backend/config/`: `config.yaml` (Standard),
`config.dev.yaml` (Debug, großzügige Limits), `config.prod.yaml` (Port 443,
eigene TLS-Zertifikate, private IPs gesperrt).

---

## 5. Sicherheit & Datenschutz

DocFlow ist Privacy-by-design: Dokumente und Bilder existieren ausschließlich
im Arbeitsspeicher, es wird nichts auf die Festplatte geschrieben und nichts
irgendwohin gesendet - außer an die eigenen Geräte im LAN.

| Massnahme | Umsetzung |
|-----------|-----------|
| RAM-only | Bilder und PDF leben nur im Speicher; nach dem Download wird alles freigegeben |
| Burn-after-Reading | Nach dem PDF-Download wird die Session inkl. Bilder sofort gelöscht |
| Session-Timeout | Inaktive Sessions werden spätestens nach 1h (konfigurierbar) bereinigt |
| PIN | 6-stellig, `crypto/rand`, constant-time Vergleich, Sperre nach Fehlversuchen |
| WebSocket-Auth | Die PIN wird als erste Nachricht gesendet - niemals in URLs (sonst würde sie in Logs landen) |
| HTTPS | Automatisch generiertes TLS-Zertifikat oder eigene Zertifikate via Config |
| Access-Logs | Ohne IP-Adressen, ohne Query-Strings, ohne PIN |
| Upload-Begrenzung | `io.LimitReader` erzwingt das konfigurierte Größenlimit |
| Security-Header | HSTS, CSP, X-Frame-Options u.a. |
| Rate-Limiting | Konfigurierbares Limit pro IP für API-Endpunkte |

Die vollständige Datenschutz-Erklärung: [PRIVACY.md](PRIVACY.md) (Deutsch)
bzw. [PRIVACY.en.md](PRIVACY.en.md) (Englisch).

---

## 6. Fehlerbehebung

| Problem | Lösung |
|---------|--------|
| Zertifikatswarnung im Browser | Erwartet bei selbstsigniertem Zertifikat - einmalig akzeptieren oder eigene Zertifikate via `server.tls_cert_path`/`tls_key_path` einbinden |
| Handy erreicht den Server nicht | Desktop und Handy müssen im selben LAN sein; Firewall-Port prüfen; QR-Code zeigt automatisch die erkannte LAN-IP an |
| PIN wird abgelehnt, obwohl korrekt | Nach 3 Fehlversuchen ist die Session gesperrt - die Sperrdauer (Standard 5 min) abwarten oder eine neue Session starten |
| Kipp-Indikator reagiert nicht | Der Browser blockiert möglicherweise Sensoren - DocFlow wechselt automatisch auf Bildanalyse; ggf. Kamera-Berechtigung erteilen |
| Upload schlägt fehl | Bildgröße oder Seitenzahl überschritten (Standard: 10 MB/Bild, 20 Seiten) |
| Session plötzlich weg | Session-Timeout abgelaufen (Standard 1 h) - einfach eine neue Session erstellen |
| Kein Bild-Update auf dem Desktop | WebSocket-Verbindung prüfen; die PIN wird als erste Nachricht gesendet, ein Proxy darf die Verbindung nicht umbrechen |

---

## 7. Grenzen des Systems

- Bis zu 20 Seiten pro Session und 10 MB pro Bild (konfigurierbar)
- JPEG-Qualität 85% im PDF (konfigurierbar)
- Single-Instance-Betrieb: Sessions liegen im RAM, ein Neustart verwirft alle
  laufenden Sessions
- Konzipiert für kleine Gruppen im lokalen Netzwerk, nicht für
  Multi-Server-Betrieb
