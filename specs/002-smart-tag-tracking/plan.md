# Implementation Plan: Schema-Aware Tag Matching & Explicit Channel Tracking

**Git Branch**: `feat/smart-tag-tracking` (base: `develop`) | **Spec Dir**: `specs/002-smart-tag-tracking/` | **Date**: 2026-11-22 | **Spec**: [spec.md](file:///Users/giacomo/dev/workspaces/container-updater/specs/002-smart-tag-tracking/spec.md)

**Input**: Feature specification from `/specs/002-smart-tag-tracking/spec.md` (tracking issue [#38](https://github.com/giacomoRanieri/container-updater/issues/38)).

## Summary

Eliminate false-positive image tag matches (e.g. CalVer `2026.11.3` vs SemVer `3.11.4`, compact dates `20230811` vs `2.48.7`, `-alpine` misclassified as prerelease) by replacing the naive version parser in `internal/registry/semver.go` with a schema-, segment-, prefix-, flavor-, and revision-aware version matcher (Livello 1). Additionally, introduce explicit opt-in channel tracking (`container-updater.track-tag` and K8s per-container `container-updater.<container>.track-tag`) across OCI config label inspection, K8s/Docker discovery, SQLite persistence with non-destructive schema migration, monitor update checks with downgrade protection, Apprise notifications, and a read-only Web UI badge (Opzione A).

## Technical Context

**Language/Version**: Go 1.25+ (`backend/`), TypeScript 5 / React 19 / Next.js 16 Pages Router (`frontend/`)

**Primary Dependencies**: `net/http`, `encoding/json`, `modernc.org/sqlite`, `k8s.io/client-go`, `github.com/docker/docker/client`

**Storage**: SQLite (`workloads` table with additive `track_tag TEXT` column and idempotent `PRAGMA table_info` migration)

**Testing**: `go test -race -count=1 ./...`, `go vet ./...`, `CGO_ENABLED=0 GOOS=linux go build`, `pnpm lint`, `npx tsc --noEmit`, `pnpm build`

**Target Platform**: Linux multi-arch containers (`linux/amd64`, `linux/arm64`) for Docker Compose and Kubernetes

**Project Type**: Multi-component web application (`backend/` Go daemon + `frontend/` Next.js Pages Router app)

**Performance Goals**: Channel-tracked workloads aligned with their target channel skip `/v2/*/tags/list` pagination entirely (single `HEAD`/digest comparison).

**Constraints**: Pure-Go static binary (`CGO_ENABLED=0`), no new external dependencies, read-only YAML-driven channel config in UI.

## Constitution Check

Verify alignment with [Container Updater Constitution](../../.specify/memory/constitution.md) (v1.0.0):

- [x] **Principle I (Decoupled Architecture & Pure-Go Runtime)**: Backend and frontend communicate exclusively via REST/JSON (`Workload.track_tag`); pure Go with zero CGO dependencies.
- [x] **Principle II (Safe Manifest Mutation & GitOps Integrity)**: Tracked channel resolution resolves pinned version tags (`repo:2.48.8`) so existing YAML AST updaters preserve pinned tags and never replace them with floating `:stable` unless no versioned tag exists.
- [x] **Principle III (Security-First Authentication & Secret Handling)**: OCI token auth stays isolated in `registry.Client`; Go default redirect handling prevents leaking `Authorization` headers across CDN/blob redirects; no secrets logged or added.
- [x] **Principle IV (Regression-First Testing & Static Quality Gates)**: Regression/unit tests planned in `internal/registry`, `internal/db`, `internal/k8s`, `internal/docker`, and `internal/monitor`; full backend and frontend CI checks executed.
- [x] **Principle V (Observability, Auditability & Simplicity)**: Uses `internal/logger` structured logging, YAGNI design with small helpers and zero speculative frameworks.
- [x] **Git Governance**: Branched `feat/smart-tag-tracking` from `develop`, Conventional Commits referencing `#38`, PR target `develop`.

## Project Structure

```text
specs/002-smart-tag-tracking/
├── spec.md              # Feature specification
├── plan.md              # This implementation plan
└── tasks.md             # Ordered task checklist

backend/internal/
├── registry/
│   ├── semver.go        # Livello 1: Schema, flavor, prefix, segment & ls-revision matcher
│   ├── semver_test.go   # Comprehensive real-world regression suite
│   ├── client.go        # Opzione A: FetchImageLabels + authenticated GET helper
│   └── client_test.go   # httptest OCI index/attestation/blob unit tests
├── db/
│   ├── db.go            # Add track_tag column + idempotent migrateSchemas()
│   ├── workload.go      # Workload.TrackTag field & CRUD queries
│   └── workload_test.go # Persistence & legacy table migration tests
├── k8s/
│   ├── client.go        # resolveTrackTag (per-container + workload + pod fallback)
│   └── client_test.go   # Annotation precedence tests
├── docker/
│   ├── client.go        # resolveDockerTrackTag label extraction
│   └── client_test.go   # Label extraction tests
└── monitor/
    ├── check.go         # registryAPI interface, explicit track-tag resolution & downgrade guard
    ├── check_test.go    # Mocked OCI check scenarios & regression tests
    └── scheduler.go     # Notification enrichment (NewImage, TrackTag) & re-notify on tag change

frontend/src/
├── services/api.ts              # Workload.track_tag?: string
└── components/WorkloadItem.tsx  # Safe extractImageTag + read-only Tracked Channel badge

README.md                        # Document matching behavior & track-tag labels/annotations
```

## Complexity Tracking

No Constitution violations. All additions reuse existing package boundaries and standard library primitives.
