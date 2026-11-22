<!--
Sync Impact Report:
- Version change: template -> 1.0.0 (initial ratification)
- Modified principles:
  - PRINCIPLE_1_NAME -> I. Decoupled Architecture & Pure-Go Runtime
  - PRINCIPLE_2_NAME -> II. Safe Manifest Mutation & GitOps Integrity
  - PRINCIPLE_3_NAME -> III. Security-First Authentication & Secret Handling
  - PRINCIPLE_4_NAME -> IV. Regression-First Testing & Static Quality Gates
  - PRINCIPLE_5_NAME -> V. Observability, Auditability & Simplicity
- Added sections:
  - Technical Stack & Platform Constraints
  - Development Workflow & Git Governance
- Removed sections: None
- Templates & workflow artifacts updated:
  - ✅ .specify/memory/constitution.md
  - ✅ .specify/templates/spec-template.md
  - ✅ .specify/templates/plan-template.md
  - ✅ .specify/templates/tasks-template.md
  - ✅ .specify/scripts/bash/create-new-feature.sh
  - ✅ CONTRIBUTING.md
- Follow-up TODOs: None
-->

# Container Updater Constitution

## Core Principles

### I. Decoupled Architecture & Pure-Go Runtime

The Go backend (`backend/`, Go 1.25+, Chi REST router, and Gorilla WebSocket hub) MUST remain
strictly decoupled from the Next.js frontend (`frontend/`, Next.js 16, React 19, TypeScript, and
Vanilla CSS). All backend-frontend interactions MUST occur through documented REST and WebSocket
contracts without shared runtime state. The backend binary MUST compile statically with
`CGO_ENABLED=0` using the pure-Go SQLite driver (`modernc.org/sqlite`) to preserve cross-platform
multi-architecture portability (`linux/amd64` and `linux/arm64`) and keep daemon memory overhead
low (under 50MB RAM target).

### II. Safe Manifest Mutation & GitOps Integrity

Docker Compose and Kubernetes YAML manifest updates MUST mutate only the targeted container image
reference while preserving multi-document YAML separators (`---`), original indentation, and AST
comments via `gopkg.in/yaml.v3` AST manipulation. Concurrent update requests targeting the same
workload MUST be rejected or serialized to prevent race conditions and partial writes. When GitOps
synchronization is enabled, automated commits and pushes MUST target only modified manifest files,
MUST NOT discard unrelated configuration, and MUST leave workspaces clean after completion or error.

### III. Security-First Authentication & Secret Handling

External Web UI and API access MUST enforce OIDC/OAuth2 authentication when `OIDC_ENABLED=true`,
with mock `admin` authentication strictly opt-in (`OIDC_ENABLED=false`) for local development.
Sensitive credentials—including registry passwords and access tokens, Apprise notification URLs,
GitOps SSH keys or HTTP PATs, and `GITHUB_TOKEN`—MUST be masked in API responses, UI views, and
structured logs, and MUST NEVER be persisted or logged in cleartext outside secure storage. Runtime
frontend configuration (such as `NEXT_PUBLIC_BACKEND_URL`) MUST be injected dynamically via
`/env-config.js` at container startup rather than baked into build-time static assets.

### IV. Regression-First Testing & Static Quality Gates

Critical backend packages (`internal/k8s`, `internal/docker`, `internal/registry`, `internal/git`,
`internal/db`, and `internal/api`) MUST maintain automated Go unit tests executed under the race
detector (`go test -race ./...`). Every bugfix MUST include a regression test reproducing the defect
before or alongside the fix. Before any Pull Request can merge, the backend MUST pass `go vet ./...`,
`go test -race -count=1 ./...`, and a static `CGO_ENABLED=0` build, while the frontend MUST pass
`pnpm lint`, `npx tsc --noEmit`, and `pnpm build`. Contributors editing Next.js code MUST consult the
bundled version guides in `frontend/node_modules/next/dist/docs/` per `frontend/AGENTS.md`.

### V. Observability, Auditability & Simplicity

The backend MUST emit structured JSON logs via `internal/logger` (`log/slog`), persist every update
job outcome (workload identifier, status, timestamps, and error details) in the SQLite audit log,
and stream live rollout and job progress events over the WebSocket hub. Features and refactoring MUST
follow YAGNI: avoid speculative abstractions, unnecessary third-party frameworks, or architectural
complexity unless explicitly justified in the plan's Complexity Tracking table.

## Technical Stack & Platform Constraints

- **Backend**: Go 1.25+ (`backend/cmd/updater`, `backend/internal/*`), Chi v5, Gorilla WebSocket,
  Docker Engine API client, Kubernetes `client-go` v0.30.x, `go-git/v5`, `robfig/cron/v3`, `coreos/go-oidc/v3`,
  `gopkg.in/yaml.v3`, and `modernc.org/sqlite`.
- **Frontend**: Node.js 24 (CI/Docker runtime) / 20+ (local), `pnpm` 10 / Corepack, Next.js 16 (Pages
  Router), React 19, TypeScript 5, ESLint 9 (`eslint-config-next`), and Vanilla CSS.
- **Notifications & Integrations**: Apprise HTTP API (`APPRISE_API_URL`) for Markdown alerts and
  GitHub release changelog lookups.
- **Container & Deployment Targets**: Alpine-based static backend image (`Dockerfile.backend`) and
  Node-based frontend image (`Dockerfile.frontend`) published to GHCR as multi-arch manifests
  (`linux/amd64`, `linux/arm64`) for Docker Compose and Kubernetes workloads.

## Development Workflow & Git Governance

- **Two-Speed Development Cycle**:
  - *New features & architectural changes* MUST follow the full Spec Kit SDD cycle (`/speckit-specify`
    -> `/speckit-clarify` -> `/speckit-plan` -> `/speckit-tasks` -> `/speckit-analyze` ->
    `/speckit-implement` -> `/speckit-converge`).
  - *Targeted bugfixes & minor refinements* MAY follow a lightweight Regression-First cycle (reproducing
    test -> minimal fix -> full verification), updating `specs/*/spec.md` requirements only when
    expected functional behavior changes.
- **Modified Git Flow & Spec Kit Alignment**:
  - Direct pushes to `master` and `develop` are strictly prohibited.
  - Working branches MUST be created from `develop` using Conventional prefixes (`feat/*`, `fix/*`,
    `docs/*`, `chore/*`, `ci/*`, `refactor/*`, `test/*`) and MUST target `develop` via Pull Request.
  - Spec Kit directories (`specs/<NNN>-<short-name>/`, tracked in `.specify/feature.json`) and Git
    branch names (`feat/<short-name>`) are intentionally decoupled so SDD specs integrate cleanly with
    Git Flow without generating non-standard branch names.
- **Commit, CI & Release Gates**:
  - All commits and PR titles MUST follow Conventional Commits, enforced by `commitlint`.
  - PRs require passing CI checks (`Validate Conventional Commits`, `Backend Go Test & Vet`,
    `Frontend Lint, Typecheck & Build`, and `Multi-Arch Build and GHCR Publish`) plus at least 1
    approval from a Maintainer listed in `.github/CODEOWNERS`.
  - Releases and changelogs are automated by Release-Please (`develop` -> `master`).

## Governance

This Constitution supersedes ad-hoc development practices across the repository. Every Implementation
Plan (`Constitution Check` section) and Pull Request review MUST verify compliance with Principles
I–V and the Development Workflow & Git Governance rules. Any intentional deviation MUST be documented
and justified in the plan's `Complexity Tracking` table prior to implementation.

Amendments follow Semantic Versioning (`MAJOR` for principle removals or incompatible redefinitions,
`MINOR` for new principles or expanded guidance, `PATCH` for non-semantic clarifications), MUST be
proposed via Pull Request targeting `develop`, and MUST synchronize dependent templates in
`.specify/templates/` and documentation in `CONTRIBUTING.md`.

**Version**: 1.0.0 | **Ratified**: 2026-11-22 | **Last Amended**: 2026-11-22
