---

description: "Task list for schema-aware version tag matching and explicit OCI channel pointer tracking"
---

# Tasks: Schema-Aware Tag Matching & Explicit Channel Tracking

**Input**: Design documents from `/specs/002-smart-tag-tracking/` (`spec.md`, `plan.md`)

**Tracking Issue**: [#38](https://github.com/giacomoRanieri/container-updater/issues/38)

## Phase 1: Setup & Governance

- [x] T001 Align GitHub issues (retroactive historical features #33–#35, future backlog #36–#37, current feature #38)
- [x] T002 Sync `develop` and create `feat/smart-tag-tracking` branch + `specs/002-smart-tag-tracking/` via Spec Kit script
- [x] T003 Author `spec.md`, `plan.md`, and `tasks.md` in `specs/002-smart-tag-tracking/`

---

## Phase 2: User Story 1 — Schema- & Flavor-Aware Version Tag Matching (Livello 1)

- [x] T004 [US1] Implement `VersionScheme`, `Version` fields (`HasVPrefix`, `Parts`, `Flavor`, `HasRevision`, `Revision`), `IsCompatibleWith`, updated `GreaterThan`, and `FindMatchingTagsDescending` in `backend/internal/registry/semver.go`
- [x] T005 [US1] Add comprehensive regression tests for `matterbridge`, `n8n`, `mosquitto`, `apprise`, `victoriametrics`, `home-assistant`, `linuxserver/wireguard`, single-major, compact-date, and prerelease tags in `backend/internal/registry/semver_test.go`

---

## Phase 3: User Story 2 — Explicit OCI Channel Pointer Tracking (Opzione A)

- [x] T006 [P] [US2] Implement `FetchImageLabels` (multi-arch index + attestation skip + Bearer auth + config blob labels) in `backend/internal/registry/client.go`
- [x] T007 [P] [US2] Add `httptest` OCI manifest/index/blob unit tests for `FetchImageLabels` in `backend/internal/registry/client_test.go`
- [x] T008 [P] [US2] Add `TrackTag` to `db.Workload` and update `SaveWorkload`/`GetWorkload`/`ListWorkloads` in `backend/internal/db/workload.go`
- [x] T009 [P] [US2] Add `track_tag TEXT` schema column and idempotent `migrateSchemas()` check in `backend/internal/db/db.go`
- [x] T010 [US2] Add DB persistence and legacy-table migration unit tests in `backend/internal/db/workload_test.go`
- [x] T011 [P] [US2] Add `resolveTrackTag` with per-container override and pod-template fallback in `backend/internal/k8s/client.go` and tests in `backend/internal/k8s/client_test.go`
- [x] T012 [P] [US2] Add `resolveDockerTrackTag` in `backend/internal/docker/client.go` and tests in `backend/internal/docker/client_test.go`
- [x] T013 [US2] Refactor `backend/internal/monitor/check.go` with `registryAPI` interface, `splitImageRepoTag`, explicit `track-tag` resolution (OCI `org.opencontainers.image.version` + `ListTags` fallback + downgrade guard), and skip `ListTags` for non-version current tags
- [x] T014 [US2] Add comprehensive unit tests with fake `registryAPI` in `backend/internal/monitor/check_test.go`

---

## Phase 4: User Story 3 — Read-Only UI Visibility, Notifications & Docs

- [x] T015 [P] [US3] Enrich Apprise notification payload (`NewImage`, `TrackTag`) and re-notify on `NewImage` change in `backend/internal/monitor/scheduler.go`
- [x] T016 [P] [US3] Add `track_tag?: string` to `Workload` interface in `frontend/src/services/api.ts`
- [x] T017 [P] [US3] Add safe `extractImageTag` helper and read-only `Tracked Channel` badge in `frontend/src/components/WorkloadItem.tsx`
- [x] T018 [P] [US3] Document schema-aware matching and `container-updater.track-tag` / per-container K8s annotations in `README.md`

---

## Phase 5: Verification & Git Flow Polish

- [x] T019 Run backend CI gates (`cd backend && go vet ./... && go test -race -v -count=1 ./... && CGO_ENABLED=0 GOOS=linux go build -o /dev/null ./cmd/updater`)
- [x] T020 Run frontend CI gates (`cd frontend && pnpm lint && npx tsc --noEmit && pnpm build`)
- [x] T021 Mark completed tasks, review `git diff`, and create Conventional Commits on `feat/smart-tag-tracking` referencing `#38`
