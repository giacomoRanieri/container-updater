# Implementation Plan: Container Updater

**Branch**: `001-container-updater` | **Date**: 2026-07-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-container-updater/spec.md`

## Summary

The `container-updater` application will consist of a **Go backend** acting as a background daemon (monitoring Docker/Compose and Kubernetes, updating containers/workloads, and sending notifications via Apprise) and hosting a REST/WebSocket API. It will connect to a local **SQLite database** to store UI-driven configurations and job states. The frontend will be built with **Next.js (React)**, presenting a dashboard to view status, trigger updates, and manage settings, styled with modern Vanilla CSS.

## Technical Context

**Language/Version**: Go 1.22+ (Backend), TypeScript / Node.js 20+ / pnpm 9+ (Frontend)


**Primary Dependencies**:
- Backend:
  - `github.com/docker/docker/client` (Docker Engine API Client)
  - `k8s.io/client-go` (Kubernetes API Client)
  - `modernc.org/sqlite` (Pure-Go SQLite driver, avoiding CGO)
  - `github.com/robfig/cron/v3` (Scheduler library)
  - HTTP router (e.g., `github.com/go-chi/chi/v5` or standard library `net/http`)
- Frontend:
  - `next` / `react` / `react-dom` (Next.js framework)
- Notifications:
  - Apprise (invoked via an Apprise API service, Apprise CLI wrapper, or Apprise HTTP API webhook)

**Storage**: SQLite database (using local file storage)

**Testing**: Go standard library testing package (`testing`), Jest + React Testing Library (Frontend)

**Target Platform**: Linux server, running as a Docker container with host mounts (`/var/run/docker.sock`, `/app/data/`, etc.) or deployed as a Pod in Kubernetes.

**Project Type**: Multi-component Web Service (Go daemon + Next.js Web App)

**Performance Goals**:
- Background check processes execute concurrently.
- Scanning 50 workloads takes under 15 seconds.
- Memory footprint of the Go backend daemon remains under 50MB RAM.

**Constraints**:
- Access to the host's Docker socket and/or local Kubernetes API (in-cluster configuration).
- Pure-Go SQLite configuration to simplify cross-compilation inside Docker images.

**Scale/Scope**: Up to 100 monitored workloads per instance, single cluster or host Docker engine.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No specific custom principles are defined in the constitution template. Default architectural rules apply:
1. Separation of concerns: Go backend handles monitoring/Docker API/K8s API/DB, Next.js handles user presentation.
2. Technology-agnostic REST/WebSocket endpoints link the two components.

## Project Structure

### Documentation (this feature)

```text
specs/001-container-updater/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

We will use the **Web application** project structure:

```text
backend/
├── cmd/
│   └── updater/
│       └── main.go      # Main entry point for the daemon and API server
├── internal/
│   ├── api/             # REST and WebSocket handlers
│   ├── config/          # Configurations (file-based and DB-based)
│   ├── db/              # SQLite schemas and queries
│   ├── docker/          # Docker Engine API client wrapper
│   ├── k8s/             # Kubernetes client-go client wrapper
│   ├── monitor/         # Monitoring cron runner and scheduler
│   └── notify/          # Apprise notification dispatcher
├── go.mod
└── go.sum

frontend/
├── src/
│   ├── components/      # UI components (Dashboard, Settings, Logs, Auth)
│   ├── pages/           # Next.js page routes
│   ├── styles/          # Vanilla CSS modules and design tokens
│   └── services/        # Frontend API client
├── package.json
└── tsconfig.json

Dockerfile.backend       # Multi-stage build for Go backend daemon
Dockerfile.frontend      # Multi-stage build for Next.js frontend
docker-compose.yml       # Docker Compose definition for containerized run

```

**Structure Decision**: Multi-component workspace. The backend directory will contain the Go application, and the frontend directory will contain the Next.js TypeScript application.

## Complexity Tracking

*No violations identified.*
