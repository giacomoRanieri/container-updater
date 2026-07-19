# Tasks: Container Updater

**Input**: Design documents from `/specs/001-container-updater/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/api.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Web app project structure**: Go backend under `backend/src/` (or package structure) and Next.js frontend under `frontend/src/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create project directories for `backend/` and `frontend/`
- [x] T002 Initialize Go modules in `backend/go.mod`
- [x] T003 [P] Initialize Next.js project using pnpm in `frontend/package.json`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Setup SQLite database initialization and schema creation in `backend/internal/db/db.go`
- [x] T005 Implement environment variables configuration in `backend/internal/config/config.go`
- [x] T006 Setup basic HTTP routing and middleware framework in `backend/internal/api/router.go`
- [x] T007 Configure structured logging and global error handling in `backend/internal/logger/logger.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Automatic Update Monitoring & Notifications (Priority: P1) 🎯 MVP

**Goal**: Periodically check container versions and send notifications via Apprise when updates are found.

**Independent Test**: Run check interval, verify registry comparison, and confirm Telegram/HA notifications are received.

### Implementation for User Story 1

- [x] T008 [P] [US1] Create Workload entity model and SQLite CRUD queries in `backend/internal/db/workload.go`
- [x] T009 [P] [US1] Create NotificationService entity model and SQLite CRUD queries in `backend/internal/db/notification.go`
- [x] T010 [US1] Implement Docker client wrapper to list active containers filtered by label in `backend/internal/docker/client.go`
- [x] T011 [US1] Implement Kubernetes client wrapper to list active workloads filtered by annotation in `backend/internal/k8s/client.go`
- [x] T012 [US1] Implement Apprise notification dispatcher client in `backend/internal/notify/apprise.go`
- [x] T013 [US1] Implement registry check and comparison logic (SHA digest lookup) with private registry auth support in `backend/internal/monitor/check.go`
- [x] T014 [US1] Implement GitHub repository metadata extraction and Changelog fetching in `backend/internal/notify/changelog.go`
- [x] T015 [US1] Implement cron scheduler to run background checks at intervals in `backend/internal/monitor/scheduler.go`

**Checkpoint**: At this point, User Story 1 is fully functional and testable independently.

---

## Phase 4: User Story 2 - Web UI Dashboard & Authentication (Priority: P2)

**Goal**: Display monitored workloads and audit logs on a secure React dashboard.

**Independent Test**: Load page, verify OIDC login redirect, and view workloads list and audit log history.

### Implementation for User Story 2

- [x] T016 [US2] Implement OIDC OAuth2 login, callback, and session REST endpoints in `backend/internal/api/auth.go`
- [x] T017 [US2] Implement REST endpoints to list workloads, audit logs, and app stats in `backend/internal/api/workloads.go`
- [x] T018 [US2] Implement WebSocket server for real-time workload notifications in `backend/internal/api/websocket.go`
- [x] T019 [P] [US2] Implement OIDC authentication redirection and session client in `frontend/src/services/auth.ts`
- [x] T020 [P] [US2] Implement API service client to fetch workloads, audit logs, and handle WebSocket streams in `frontend/src/services/api.ts`
- [x] T021 [US2] Implement design tokens and CSS base layout (Vanilla CSS) in `frontend/src/styles/globals.css`
- [x] T022 [P] [US2] Create Dashboard workload list component in `frontend/src/components/Dashboard.tsx`
- [x] T023 [P] [US2] Create Audit Log / Job History list component in `frontend/src/components/AuditLog.tsx`
- [x] T024 [US2] Create login page and dashboard layout in `frontend/src/pages/index.tsx`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently.

---

## Phase 5: User Story 3 - Manual Update Execution (Priority: P1)

**Goal**: Click an "Update" button to update a container/workload.

**Independent Test**: Click button, verify container restart/k8s deployment update, and confirm update status in UI.

### Implementation for User Story 3

- [x] T025 [US3] Create UpdateJob entity model and SQLite queries in `backend/internal/db/job.go`
- [x] T026 [US3] Implement Compose YAML modification and shell execution of `docker compose up -d` in `backend/internal/docker/update.go`
- [x] T027 [US3] Implement Kubernetes YAML manifest modification and client-go application in `backend/internal/k8s/update.go`
- [x] T028 [US3] Implement REST endpoint to trigger manual workload update job in `backend/internal/api/jobs.go`
- [x] T029 [P] [US3] Add manual update trigger buttons to the dashboard component in `frontend/src/components/WorkloadItem.tsx`

**Checkpoint**: All user stories should now be independently functional.

---

## Phase 6: User Story 4 - GitOps Git Integration (Priority: P3)

**Goal**: Commit and push manifest updates to a Git repository.

**Independent Test**: Verify file is updated, committed, and pushed to remote Git repository.

### Implementation for User Story 4

- [x] T030 [US4] Implement GitOps Git repository clone, commit, and push wrapper in `backend/internal/git/gitops.go`
- [x] T031 [P] [US4] Add Git authentication settings (SSH keys or token) in `backend/internal/config/config.go`

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T032 [P] Add Apprise notification and Git settings page in `frontend/src/pages/settings.tsx`
- [x] T033 [P] Create multi-stage container deployment configurations in `Dockerfile.backend`, `Dockerfile.frontend`, and `docker-compose.yml`
- [x] T034 Update `README.md` with container build, volume mount, and deployment instructions
- [x] T035 Run end-to-end validation scenarios documented in `specs/001-container-updater/quickstart.md`
- [x] T036 [P] Create GitHub Actions matrix build workflow (amd64/arm64) and GHCR push without QEMU emulation in `.github/workflows/ci-cd.yml`

---

## Phase 8: User Story 5 - Repository Version Management & CI/CD Automation (Priority: P2)

**Goal**: Automate Conventional Commits linting, Release-Please SemVer PR generation, Beta/GA multi-arch GHCR image builds, and CODEOWNERS branch protection.

- [x] T037 [US5] Create CODEOWNERS configuration for maintainer approvals in `.github/CODEOWNERS`
- [x] T038 [US5] Implement Commitlint configuration and GitHub Actions workflow in `.github/commitlint.config.js` and `.github/workflows/commitlint.yml`
- [x] T039 [US5] Implement Release-Please GitHub Actions workflow for SemVer automation and CHANGELOG updates in `.github/workflows/release-please.yml`
- [x] T040 [US5] Update multi-arch CI/CD workflow to support Beta (`develop`) and GA Release (`master`/tags) Docker publishing to GHCR in `.github/workflows/ci-cd.yml`
- [x] T041 [US5] Create repository contributing and branch protection documentation in `CONTRIBUTING.md`
- [x] T042 [US5] Implement PR Preview Docker image tag generation (`:pr-<N>`) and GitHub Environment approval gate (`environment: beta-build`) in `.github/workflows/ci-cd.yml`
- [x] T043 [US5] Implement automated PR preview image tag deletion upon PR closure in `.github/workflows/pr-cleanup.yml`
- [x] T044 [US5] Document PR preview artifact lifecycle and Environment Approval Gates in `CONTRIBUTING.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Once Foundational phase completes, all user stories can start in parallel
- Models/APIs within a story marked [P] can run in parallel

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready
