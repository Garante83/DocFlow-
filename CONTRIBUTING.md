# Contributing to DocFlow

Thank you for your interest in contributing! This document provides guidelines for contributing to this project.

## Development Setup

### Prerequisites

- **Go** 1.26 or later (see `backend/go.mod`)
- **Node.js** 22.18+ or 24.12+ and npm (see `frontend/dokumentenscanner/package.json`)
- **Make** (optional, for build commands)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/your-username/DocFlow.git
cd DocFlow

# Setup Backend
cd backend
go mod download

# Setup Frontend
cd frontend/dokumentenscanner
npm install
```

## Development Workflow

### Running Locally

```bash
# Terminal 1: Backend
cd backend
make run

# Terminal 2: Frontend (with hot reload)
cd frontend/dokumentenscanner
npm run dev
```

### Code Style

**Go:**
- Run `make lint` before committing
- Follow standard Go conventions
- Use `gofmt` for formatting

**TypeScript/Vue:**
- ESLint is configured
- Run `npm run lint` to check

### Testing

```bash
# Backend tests
cd backend
make test

# Frontend tests
cd frontend/dokumentenscanner
npm run test:unit -- --run
```

All tests must pass before submitting a PR.

## Commit Guidelines

- Write clear, concise commit messages
- Reference issues when applicable (e.g., "Fixes #42")
- Keep commits focused - one logical change per commit

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests and linting
5. Commit with a clear message
6. Push to your fork
7. Open a Pull Request

### PR Description

Include:
- What the change does
- Why the change is needed
- Any breaking changes
- Test results

## Reporting Issues

- Use GitHub Issues
- Include steps to reproduce
- Include your environment (OS, browser, Go version)
- Include any error messages

## Code of Conduct

Be respectful and constructive in all interactions.

## Questions?

Open an issue for any questions about contributing.
