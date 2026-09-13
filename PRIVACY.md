# Datenschutzerklärung / Privacy Policy

*Gültig ab: 2026-09-13 (Englische Fassung: [PRIVACY.en.md](PRIVACY.en.md))*

## Verantwortlicher

Das DocFlow-Projekt ist Open-Source-Software. Wer DocFlow selbst betreibt
(selbst hostet), ist für den eigenen Betrieb **Einrichtungsverantwortlicher**
im Sinne der DSGVO. Diese Beschreibung dokumentiert die technische
Datenverarbeitung der Software.

## Verarbeitete Daten

Die Anwendung verarbeitet **Dokumentenbilder** und **PDFs**, die vom Nutzer
hochgeladen werden, sowie die technisch erforderlichen Verbindungsmetadaten:

| Datenart | Zweck | Speicherdauer |
|----------|-------|---------------|
| Dokumentenbilder | Temporäre Verarbeitung im Arbeitsspeicher | Nur während der Übertragung |
| Generiertes PDF | Temporäre Verarbeitung im Arbeitsspeicher | Nur bis zum Download |
| Session-Metadaten (UUID, PIN) | Verbindungszuordnung | Max. 1 Stunde oder bis zum Download |
| IP-Adressen (Rate-Limiting) | Missbrauchsbekämpfung | Im RAM, spätestens 10 Minuten nach letzter Anfrage gelöscht |

## Speicherort und Löschung

- **Keine persistente Speicherung:** Alle Bilder, PDFs und Sessions werden
  ausschließlich im Arbeitsspeicher (RAM) des Servers verarbeitet. Es findet
  **keine Speicherung auf Festplatten, in Datenbanken oder bei externen
  Diensten** statt.
- **Automatische Löschung nach Download:** Sobald das PDF vom Desktop
  heruntergeladen wird, werden die Bilder und das PDF aus dem Speicher
  entfernt und die Session beendet.
- **Automatische Löschung nach Timeout:** Nach Ablauf der Session-Lebensdauer
  (Standard: 1 Stunde, konfigurierbar) wird die Session inklusive aller Bilder
  gelöscht.
- **Manuelle Löschung:** Über den API-Endpunkt `DELETE /api/session/{id}`
  können Session und alle Daten jederzeit gelöscht werden.

## Clientseitige Verarbeitung auf dem Endgerät

Kamera, Bildbearbeitung (Zuschneiden, Drehen) und der Winkel-Indikator
laufen **ausschließlich im Browser des Endgeräts**: Der Kamerastream und die
Bildanalyse der Video-Frames verlassen das Gerät nicht. Nur die vom Nutzer
ausdrücklich hochgeladenen Fotos werden an den Server übertragen.

## Logging

Die Zugriffsprotokolle enthalten Methode, Pfad (ohne Query-String), Statuscode
und Antwortzeit. Session-IDs sind zufällige UUIDs ohne Bezug zu Personen.
**IP-Adressen, PINs und Dokumenteninhalte werden nicht geloggt**; die
WebSocket-PIN wird als erste WebSocket-Nachricht übertragen und taucht in
keiner URL auf.

Hinweis für Betreiber: Der Server gibt Logdaten auf stdout aus und speichert
sie selbst nicht dauerhaft. Werden sie vom Betreiber weitergeleitet
(z. B. in Logdateien), obliegt die Speicherdauer der Verantwortung des
Betreibers. Für den internen Missbrauchsschutz verarbeitet der
Rate-Limiter IP-Adressen; dies geschieht ausschließlich im RAM und wird
nicht geloggt.

## Keine Weitergabe an Dritte

Es findet **keine Weitergabe** von Daten an Dritte, keine Analyse und kein
Tracking statt. Die Anwendung lädt **keine externen Ressourcen** (Fonts,
Skripte o. Ä.) von Drittanbietern - alle Inhalte werden vom eigenen Server
ausgeliefert.

## Verschlüsselung

Die gesamte Kommunikation erfolgt über **HTTPS/TLS-verschlüsselte
Verbindungen**. Standardmäßig nutzt DocFlow ein automatisch generiertes
selbstsigniertes Zertifikat, das beim ersten Besuch manuell bestätigt werden
muss. Für produktive Installationen können eigene, vertrauenswürdige
Zertifikate über die Konfiguration (`tls_cert_path`/`tls_key_path`)
hinterlegt werden.

## Rechte der betroffenen Personen

- **Recht auf Löschung (Art. 17 DSGVO):** Session-Daten werden über
  `DELETE /api/session/{id}` gelöscht oder automatisch nach Download/Timeout.
  Da keine Daten persistent gespeichert werden, endet jede Löschung im RAM.
- **Recht auf Auskunft (Art. 15 DSGVO):** Aufgrund der in dieser Erklärung
  beschriebenen, temporären RAM-Verarbeitung liegen dem Betreiber
  dauerhaft ablegbare Daten vor, die den Umfang einer Auskunft praktisch
  begrenzen. Die verarbeiteten Daten (eigene Dokumente) sind dem Nutzer
  ohnehin bekannt.
- **Recht auf Datenübertragbarkeit (Art. 20 DSGVO):** Das generierte PDF
  steht dem Nutzer direkt als Download zur Verfügung.

## Technische Details

- **PIN-Speicherung:** Die 6-stellige PIN wird ausschließlich im RAM
  gespeichert, nie auf Festplatte geschrieben und in keiner Logdatei
  protokolliert. Der PIN-Vergleich erfolgt gegen Timing-Angriffe
  geschützt (zeitkonstant).
- **Rate-Limiting:** Zur Missbrauchsbekämpfung wird eine IP-basierte
  Anfragebegrenzung eingesetzt (siehe Tabelle oben).

## Technische Transaktion

Die gesamte Verarbeitung dient ausschließlich der technischen Übertragung
der Dokumente zwischen Desktop und Mobilgerät. Es findet keine
zweckentfremdete Verarbeitung statt.

---

*Hinweis: Diese Erklärung beschreibt die technische Datenverarbeitung der
Software korrekt nach Stand des Quellcodes. Für den produktiven Betrieb wird
eine rechtliche Prüfung durch qualifiziertes Personal empfohlen.*
