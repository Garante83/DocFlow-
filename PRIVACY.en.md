# Privacy Policy / Datenschutzerklärung

*Valid from: 2026-09-13 (German version: [PRIVACY.md](PRIVACY.md))*

## Controller

The DocFlow project is open-source software. Anyone who operates DocFlow
themselves (self-hosting) is the **controller** ("Einrichtungsverantwortlicher")
within the meaning of the GDPR for their own operation. This document describes
the technical data processing of the software.

## Data Processed

The application processes **document images** and **PDFs** uploaded by the user
as well as the technically required connection metadata:

| Data type | Purpose | Retention |
|-----------|---------|-----------|
| Document images | Temporary processing in main memory | Only during transfer |
| Generated PDF | Temporary processing in main memory | Only until download |
| Session metadata (UUID, PIN) | Session linkage | Max. 1 hour or until download |
| IP addresses (rate limiting) | Abuse prevention | In RAM, deleted at the latest 10 minutes after the last request |

## Storage Location and Deletion

- **No persistent storage:** All images, PDFs and sessions are processed
  exclusively in the server's main memory (RAM). There is **no storage on hard
  disks, in databases or with external services**.
- **Automatic deletion after download:** As soon as the PDF is downloaded by
  the desktop client, the images and the PDF are removed from memory and the
  session ends.
- **Automatic deletion after timeout:** After the session lifetime expires
  (default: 1 hour, configurable), the session including all images is deleted.
- **Manual deletion:** The session and all data can be deleted at any time via
  the API endpoint `DELETE /api/session/{id}`.

## Client-Side Processing on the Device

Camera, image editing (crop, rotate) and the angle indicator run **exclusively
in the browser of the device**: the camera stream and the video-frame analysis
never leave the device. Only photos explicitly uploaded by the user are sent to
the server.

## Logging

The access logs contain method, path (without query string), status code and
response time. Session IDs are random UUIDs with no link to a person.
**IP addresses, PINs and document contents are never logged**; the WebSocket
PIN is transmitted as the first WebSocket message and never appears in any URL.

Note for operators: The server writes log data to stdout and does not persist
them itself. If the operator forwards them (e.g. to log files), retention is
the operator's responsibility. For internal abuse protection, the rate limiter
processes IP addresses; this happens exclusively in RAM and is never logged.

## No Disclosure to Third Parties

There is **no disclosure** of data to third parties, no analytics and no
tracking. The application loads **no external resources** (fonts, scripts or
similar) from third-party providers - all content is served by the operator's
own server.

## Encryption

All communication uses **HTTPS/TLS-encrypted connections**. By default, DocFlow
uses an automatically generated self-signed certificate that must be manually
accepted on first visit. For production deployments, own, trusted certificates
can be configured (`tls_cert_path`/`tls_key_path`).

## Rights of the Data Subjects

- **Right to erasure (Art. 17 GDPR):** Session data is deleted via
  `DELETE /api/session/{id}` or automatically after download/timeout. Since no
  data is stored persistently, every erasure ends in RAM.
- **Right of access (Art. 15 GDPR):** Given the temporary RAM-only processing
  described in this document, there is practically no persistently stored data
  available to the operator that would be subject to an access request. The
  processed data (the user's own documents) is known to the user anyway.
- **Right to data portability (Art. 20 GDPR):** The generated PDF is directly
  available to the user as a download.

## Technical Details

- **PIN storage:** The 6-digit PIN is stored exclusively in RAM, never written
  to disk and never written to any log file. PIN comparison is protected
  against timing attacks (constant-time).
- **Rate limiting:** An IP-based request limiter is used for abuse prevention
  (see table above).

## Technical Transaction

The entire processing serves exclusively the technical transfer of documents
between desktop and mobile device. There is no processing beyond this purpose.

---

*Note: This policy accurately describes the technical data processing of the
software as of the current source code. For production operation, a legal
review by qualified personnel is recommended.*
