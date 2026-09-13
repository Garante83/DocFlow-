# Datenschutzerklärung / Privacy Policy

*Gültig ab: 2026-07-25*

## Verantwortlicher

Das DocFlow-Projekt ist eine Open-Source-Software. Für den Betrieb ist der Einrichtungsverantwortliche zuständig.

## Verarbeitete Daten

Die Anwendung verarbeitet **Dokumentenbilder** und **PDFs**, die vom Nutzer hochgeladen werden. Es werden keine personenbezogenen Daten erhoben, die nicht für den technischen Vorgang der Dokumentenübertragung erforderlich sind.

| Datenart | Zweck | Speicherdauer |
|----------|-------|---------------|
| Dokumentenbilder | Temporäre Verarbeitung im Arbeitsspeicher | Nur während der Übertragung |
| Generiertes PDF | Temporäre Verarbeitung im Arbeitsspeicher | Nur bis zum Download |
| Session-Metadaten (UUID, PIN) | Verbindungszuordnung | Max. 1 Stunde oder bis zum Download |

## Speicherort und Löschung

- **Keine persistente Speicherung**: Alle Bilder und PDFs werden ausschließlich im Arbeitsspeicher (RAM) des Servers verarbeitet. Es findet **keine Speicherung auf Festplatten, Datenbanken oder externen Diensten** statt.
- **Automatische Löschung nach Download**: Sobald das PDF vom Desktop heruntergeladen wird, werden alle Bilder und das PDF aus dem Speicher entfernt.
- **Automatische Löschung nach Timeout**: Nach 1 Stunde Inaktivität wird die gesamte Session automatisch gelöscht.
- **Manuelle Löschung**: Über den API-Endpunkt `DELETE /api/session/{id}` können jederzeit alle Daten gelöscht werden.

## Kein Logging von Dokumenteninhalten

Es werden **keine Inhalte von Dokumenten geloggt**. Nur Session-Metadaten (Session-ID, Status, Zeitstempel) werden für Debugging-Zwecke protokolliert. **IP-Adressen** werden im Rahmen der HTTP-Logs für technische Zwecke erfasst, aber nicht dauerhaft gespeichert.

## Keine Weitergabe an Dritte

Es findet **keine Weitergabe** von Daten an Dritte, keine Analyse und kein Tracking statt. Die Anwendung lädt keine externen Ressourcen (Fonts, Skripte etc.) von Drittanbietern.

## Verschlüsselung

Die gesamte Kommunikation erfolgt über **HTTPS/TLS-verschluesselte Verbindungen**. Bilder und PDFs werden nur verschluesselt uebertragen.

## Technische Details

- **PIN-Speicherung**: Die 6-stellige PIN wird ausschließlich im RAM gespeichert und nicht auf Festplatten geschrieben.
- **Rate-Limiting**: Zur Missbrauchsbekämpfung wird eine IP-basierte Anfragenbegrenzung eingesetzt.

## Rechte des Betroffenen

- **Recht auf Löschung** (Art. 17 DSGVO): Jederzeit über `DELETE /api/session/{id}` oder automatisch nach Download/Timeout.
- **Recht auf Auskunft** (Art. 15 DSGVO): Es werden keine dauerhaften personenbezogenen Daten gespeichert, die eine Auskunft erfordern würden. Die PDF selbst wird dem Nutzer direkt bereitgestellt.
- **Recht auf Datenübertragbarkeit** (Art. 20 DSGVO): Das generierte PDF steht dem Nutzer direkt zum Download zur Verfügung.

## Technische Transaktion

Die gesamte Verarbeitung dient ausschließlich der **technischen Transaktion** der Dokumentenübertragung zwischen Desktop und Mobilgerät. Es findet keine zweckentfremdete Verarbeitung statt.
