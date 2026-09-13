# DocFlow Frontend

Vue 3 + TypeScript frontend for document scanning and PDF conversion.

## Setup

```bash
cd frontend/dokumentenscanner
npm install --legacy-peer-deps
npm run dev
```

Open `http://localhost:9000` in your browser.

## Project Structure

- `src/views/` - Main views (DesktopView, MobileView)
- `src/components/` - Reusable components (PINInput, ImageUpload, PDFPreview, QRCodeScanner)
- `src/stores/` - Pinia stores (sessionStore)
- `src/utils/` - Utilities (api.ts, websocket.ts)
- `src/router/` - Vue Router configuration

## Configuration

Edit `.env` for backend URL and ports:

```env
VITE_PORT=9000
VITE_BACKEND_URL=http://localhost:8082
VITE_FRONTEND_URL=http://localhost:9000
```

## Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run type-check` - TypeScript type checking
- `npm run lint` - Run ESLint

## Features

- PIN verification with lockout (3 attempts, 30s timeout)
- Image upload (JPEG/PNG, max 10MB) with drag & drop
- QR code scanning for mobile devices
- PDF preview and download
- WebSocket real-time updates
- Hash-based routing for compatibility
- Responsive design for desktop and mobile
