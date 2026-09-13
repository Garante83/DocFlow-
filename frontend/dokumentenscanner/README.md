# DocFlow Frontend

Vue 3 + TypeScript frontend for document scanning and PDF conversion.

## Setup

```bash
cd frontend/dokumentenscanner
npm install
npm run dev
```

Open `https://localhost:8082` in your browser (port from `.env`).

## Project Structure

- `src/views/` - Main views (DesktopView, MobileView)
- `src/components/` - Reusable components (PINInput, ImageUpload, PDFPreview, QRCodeScanner)
- `src/stores/` - Pinia stores (sessionStore)
- `src/utils/` - Utilities (api.ts, websocket.ts, angleIndicator.ts, frameAnalyzer.ts)
- `src/i18n/` + `src/locales/` - Internationalization (de/en via vue-i18n)
- `src/router/` - Vue Router configuration

## Configuration

Edit `.env` for backend URL and ports (see `.env` in this directory for
defaults; the dev server proxies nothing - API calls use relative paths when
served from the backend).

## Scripts

- `npm run dev` - Start development server with HMR
- `npm run build` - Build for production (source maps off by default)
- `npm run preview` - Preview production build
- `npm run test:unit -- --run` - Run vitest suite
- `npm run type-check` - TypeScript type checking (vue-tsc, zero errors required)
- `npm run lint:check` - Run ESLint check

## Features

- PIN verification with configurable lockout
- Multi-page image capture and editing (rotate, crop with rule-of-thirds grid)
- Camera angle indicator (gyroscope sensor with visual frame-analysis fallback)
- Image upload (JPEG/PNG/WebP, size configurable server-side)
- QR code scanning for mobile devices
- PDF preview and download
- WebSocket real-time updates (PIN auth sent as first message, never in URLs)
- Internationalization (German/English)
- Hash-based routing for compatibility
- Responsive design for desktop and mobile
