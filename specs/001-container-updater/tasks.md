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

- [ ] T001 Create project directories for `backend/` and `frontend/`
- [ ] T002 Initialize Go modules in `backend/go.mod`
- [ ] T003 [P] Initialize Next.js project in `frontend/package.json`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Setup SQLite database initialization and schema creation in `backend/internal/db/db.go`
- [ ] T005 Implement environment variables configuration in `backend/internal/config/config.go`
- [ ] T006 Setup basic HTTP routing and middleware framework in `backend/internal/api/router.go`
- [ ] T007 Configure structured logging and global error handling in `backend/internal/logger/logger.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Automatic Update Monitoring & Notifications (Priority: P1) 🎯 MVP

**Goal**: Periodically check container versions and send notifications via Apprise when updates are found.

**Independent Test**: Run check interval, verify registry comparison, and confirm Telegram/HA notifications are received.

### Implementation for User Story 1

- [ ] T008 [P] [US1] Create Workload entity model and SQLite CRUD queries in `backend/internal/db/workload.go`
- [ ] T009 [P] [US1] Create NotificationService entity model and SQLite CRUD queries in `backend/internal/db/notification.go`
- [ ] T010 [US1] Implement Docker client wrapper to list active containers in `backend/internal/docker/client.go`
- [ ] T011 [US1] Implement Kubernetes client wrapper to list active workloads in `backend/internal/k8s/client.go`
- [ ] T012 [US1] Implement Apprise notification dispatcher client in `backend/internal/notify/apprise.go`
- [ ] T013 [US1] Implement registry check and comparison logic (SHA digest lookup) in `backend/internal/monitor/check.go`
- [ ] T014 [US1] Implement cron scheduler to run background checks at intervals in `backend/internal/monitor/scheduler.go`

**Checkpoint**: At this point, User Story 1 is fully functional and testable independently.

---

## Phase 4: User Story 2 - Web UI Dashboard & Authentication (Priority: P2)

**Goal**: Display monitored workloads on a secure React dashboard.

**Independent Test**: Load page, verify OIDC login redirect, and view workloads list.

### Implementation for User Story 2

- [ ] T015 [US2] Implement OIDC OAuth2 login, callback, and session REST endpoints in `backend/internal/api/auth.go`
- [ ] T016 [US2] Implement REST endpoints to list workloads and get app stats in `backend/internal/api/workloads.go`
- [ ] T017 [US2] Implement WebSocket server for real-time workload notifications in `backend/internal/api/websocket.go`
- [ ] T018 [P] [US2] Implement OIDC authentication redirection and session client in `frontend/src/services/auth.ts`
- [ ] T019 [P] [US2] Implement API service client to fetch workloads and handle WebSocket streams in `frontend/src/services/api.ts`
- [ ] T020 [US2] Implement design tokens and CSS base layout (Vanilla CSS) in `frontend/src/styles/globals.css`
- [ ] T021 [P] [US2] Create Dashboard workload list component in `frontend/src/components/Dashboard.tsx`
- [ ] T022 [US2] Create login page and dashboard layout in `frontend/src/pages/index.tsx`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently.

---

## Phase 5: User Story 3 - Manual Update Execution (Priority: P1)

**Goal**: Click an "Update" button to update a container/workload.

**Independent Test**: Click button, verify container restart/k8s deployment update, and confirm update status in UI.

### Implementation for User Story 3

- [ ] T023 [US3] Create UpdateJob entity model and SQLite queries in `backend/internal/db/job.go`
- [ ] T024 [US3] Implement Compose service recreation logic (stop, pull, restart) in `backend/internal/docker/update.go`
- [ ] T025 [US3] Implement Kubernetes deployment image patching logic in `backend/internal/k8s/update.go`
- [ ] T026 [US3] Implement REST endpoint to trigger manual workload update job in `backend/internal/api/jobs.go`
- [ ] T027 [P] [US3] Add manual update trigger buttons to the dashboard component in `frontend/src/components/WorkloadItem.tsx`

**Checkpoint**: All user stories should now be independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T028 [P] Add Apprise notification settings configuration page in `frontend/src/pages/settings.tsx`
- [ ] T029 [P] Create multi-stage container deployment configurations in `Dockerfile.backend`, `Dockerfile.frontend`, and `docker-compose.yml`
- [ ] T030 Update `README.md` with container build and deployment instructions
- [ ] T031 Run end-to-end validation scenarios documented in `specs/001-container-updater/quickstart.md`


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
