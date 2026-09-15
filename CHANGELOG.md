# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-15

Final release: identical code base to 1.0.0-rc.1 plus two fixes found
during test-server deployment.

### Added

- **`server.public_url` configuration** (`DSCAN_SERVER_PUBLIC_URL`): the
  QR code advertises this base URL instead of the server's own view -
  essential for container deployments (the auto-detected address is the
  Docker bridge IP, unreachable from phones) and for reverse-proxy
  setups; empty keeps LAN IP auto-detection

### Fixed

- **Docker build**: backend stage base image bumped to
  `golang:1.26-alpine` (go.mod requires >= 1.26.5)
- **WebSocket origin override in docker-compose.yml** removed (scheme-less
  values could never match browser origins; built-in defaults cover
  localhost variants and private-IP origins correctly)

### Validation

- All installation methods validated on a test server (2026-09-15):
  single binary, manual build, Docker build/run (non-root), docker
  compose, and operation behind a reverse proxy with
  `DSCAN_SERVER_PUBLIC_URL`

## [1.0.0-rc.1] - 2026-09-15

Release candidate: feature-complete, all backend packages green, coverage
~85%, frontend 123/123 tests, live E2E validated on hardware (desktop +
phone over LAN, burn-after-reading, privacy logging verified).

### Added

- **Single-binary deployment**: `make release` builds the Vue frontend,
  embeds it via `//go:embed` and compiles one self-contained Go binary;
  cross-compile via `make release-linux`
- **Camera tilt indicator** (mobile): three-stage detection chain with
  automatic fallback - device orientation sensor, accelerometer, and a
  pure image-analysis fallback (grayscale segmentation + rectangle/
  perspective checks at ~4 fps) that works even when browsers block
  sensors silently
- **Multilingual UI** (vue-i18n): German and English, selected from the
  browser language
- **Dark mode**: follows the system preference (`prefers-color-scheme`);
  QR surface and PDF viewer stay white intentionally
- **Configuration system**: Viper-based with four sources (CLI flags >
  `DSCAN_*` ENV > YAML > defaults); first start auto-writes a fully
  commented `config.yaml`, existing files are never overwritten;
  read-only-filesystem fallback
- **First-start auto-configuration** incl. pre-scan of the config path
- **Rate limiting** for API endpoints (configurable, per IP, in memory)
- **Security headers** on all responses (HSTS, CSP, X-Frame-Options, ...)
- **WebSocket auth** via first message (constant-time PIN check); the PIN
  never appears in URLs or logs
- **Privacy-preserving access logging**: no IP addresses, no query
  strings, no PIN; net/http connection errors suppressed
- **Burn-after-reading**: session incl. all images is deleted after PDF
  download; RAM-only processing throughout
- **CI pipeline** (Gitea Actions): backend build/vet/gofmt/test with
  coverage, frontend typecheck/lint/vitest

### Changed

- WebSocket PIN authentication moved from query parameter to first
  message (privacy: URLs are logged by access logs and reverse proxies)
- Upload size enforcement via `io.LimitReader`
- Pin lockout after configurable failed attempts (constant-time compare)
- Frontend builds without `main.go` changes; embedded assets untracked
  in git, rebuilt on release

### Fixed

- WebSocket reconnect loop after deliberate disconnect (endless 3s
  close/reopen cycle, 1005 close storm in logs); now: deliberate
  disconnects never reconnect, real losses use exponential backoff
  (3s-48s) capped at 5 attempts
- Image orientation preserved via canvas normalization (EXIF strip +
  rotation) on every upload
- Desktop event listener leak (named handlers + `onUnmounted` cleanup)

### Security

- Constant-time PIN comparison on both HTTP and WebSocket auth paths
- PIN lockout with configurable duration
- TLS 1.2 minimum; auto-generated self-signed certificate or own certs
  via config
- No IP addresses, no query strings, no PIN in any log output

## [Unreleased]

### Ideas

- Optional PDF export of the user manual (markdown source of truth:
  `docs/manual.md`, English: `docs/manual.en.md`)
