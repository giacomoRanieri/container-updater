# Feature Specification: Schema-Aware Tag Matching & Explicit Channel Tracking

**Spec Directory**: `specs/002-smart-tag-tracking/`

**Git Branch**: `feat/smart-tag-tracking` (branched from `develop`, PR target: `develop`)

**Tracking Issue**: [#38](https://github.com/giacomoRanieri/container-updater/issues/38)

**Created**: 2026-11-22

**Status**: Implemented

**Input**: User description: "Schema-aware version tag matching and explicit OCI channel pointer tracking"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Schema- & Flavor-Aware Version Tag Matching (Priority: P1)

As a system administrator, I want `container-updater` to compare container image version tags only against compatible tags sharing the same versioning scheme, segment depth, `v`-prefix convention, and image variant/flavor suffix, so that I stop receiving false-positive update alerts for unrelated tag streams in the same repository (such as CalVer Home Assistant add-on tags on a SemVer standalone image, legacy `YYYYMMDD` compact date tags, or mismatched OS/architecture flavors).

**Why this priority**: Eliminates noisy false positives across existing workloads immediately without requiring any YAML configuration changes.

**Independent Test**: Can be fully tested in unit/regression tests for `registry.ParseVersion` and `registry.FindLatestMatchingTag` against real-world repository tag mixes (`luligu/matterbridge`, `n8nio/n8n`, `eclipse-mosquitto`, `caronc/apprise`, `victoriametrics/victoria-metrics`, `home-assistant`, `linuxserver/wireguard`).

**Acceptance Scenarios**:

1. **Given** a workload running `luligu/matterbridge:3.11.4` and a registry containing `2026.11.3`, `2026.11.3-legacy`, `alpine-3.11.4`, `dev-3.11.4`, `stable`, and `latest`, **When** `FindLatestMatchingTag` evaluates available tags, **Then** no update is detected because `2026.11.3` belongs to the CalVer scheme rather than SemVer.
2. **Given** a workload running `n8nio/n8n:2.48.7` and a registry containing `20230811`, `2.48.7-pc`, `2.48.7-amd64`, and `2.48.8-debian`, **When** `FindLatestMatchingTag` evaluates available tags, **Then** `20230811` (compact date) and suffixed variants are ignored and only a clean matching SemVer tag (e.g. `2.48.8`) triggers an update.
3. **Given** a workload running `eclipse-mosquitto:2.1.2-alpine`, **When** the registry contains `2.1.4-alpine`, `2.1-alpine`, `2.1.4-openssl`, `2.1.4`, and `2.1.5-rc1-alpine`, **Then** `-alpine` is treated as a non-prerelease flavor suffix and `2.1.4-alpine` is selected.
4. **Given** a workload running `lscr.io/linuxserver/wireguard:1.0.20260223-r0-ls117`, **When** the registry contains `1.0.20260223-r0-ls130`, **Then** the `-ls<N>` revision increment is recognized and `1.0.20260223-r0-ls130` is proposed.

---

### User Story 2 - Explicit OCI Channel Pointer Tracking via YAML `track-tag` (Priority: P1)

As a system administrator who keeps version-pinned tags in Kubernetes and Docker Compose YAML manifests (e.g. `n8nio/n8n:2.48.7`), I want to explicitly opt a workload (or an individual container in a multi-container Kubernetes pod) into tracking an OCI channel pointer tag (such as `stable` or `latest`) via declarative YAML metadata, so that I only receive update notifications when a new release is promoted to that channel (ignoring `next`/unpromoted SemVer tags like `n8nio/n8n:2.49.3`), while still resolving the exact pinned version tag to write back to my YAML.

**Why this priority**: Solves the case where pre-promoted or parallel channel releases (like `n8n:2.49.3` on `next`) carry clean SemVer strings indistinguishable from `stable` by tag name alone.

**Independent Test**: Can be tested end-to-end with mocked OCI registry responses and K8s/Docker metadata parsing: verifying that matching digests against the tracked tag mark the workload `up_to_date` without calling `ListTags`, and digest changes resolve the pinned tag via `org.opencontainers.image.version` (or digest-matching fallback) without allowing downgrades.

**Acceptance Scenarios**:

1. **Given** a workload running `n8nio/n8n:2.48.7` with `container-updater.track-tag: "stable"`, **When** the digest of `n8nio/n8n:stable` equals the digest of `2.48.7` (even if `2.49.3` exists on the registry), **Then** the workload is marked `up_to_date` and `/tags/list` is not queried.
2. **Given** a workload running `n8nio/n8n:2.48.7` with `container-updater.track-tag: "stable"`, **When** `n8nio/n8n:stable` moves to a new digest and its OCI config label `org.opencontainers.image.version` is `"2.48.8"` (matching the `stable` digest), **Then** `NewImage` is set to `n8nio/n8n:2.48.8` and `NewDigest` is set to the new digest.
3. **Given** a multi-container Kubernetes Deployment with `container-updater.wireguard.track-tag: "latest"` on container `wireguard` and no override on `wg-portal`, **When** workloads are discovered, **Then** only `wireguard` receives `TrackTag="latest"` while `wg-portal` uses default schema-aware tag matching.
4. **Given** a workload running `n8nio/n8n:2.49.3` (ahead of `stable`) with `container-updater.track-tag: "stable"` pointing to `2.48.7`, **When** the check executes, **Then** downgrade protection prevents proposing `2.48.7` and marks the workload `up_to_date`.

---

### User Story 3 - Read-Only Channel Visibility in Web UI & Notifications (Priority: P2)

As a system administrator, I want the configured tracked channel to be persisted in SQLite, displayed as a read-only indicator on workload cards in the Web UI, and included in Apprise update notifications, so that I always have clear visibility into why a workload tracks a specific release stream directly from my YAML definitions.

**Why this priority**: Keeps the YAML manifest as the single source of truth while giving full observability in the dashboard and notifications.

**Independent Test**: Can be verified via DB schema/migration tests, scheduler notification formatting, and frontend TypeScript/Next.js build and UI rendering checks.

**Acceptance Scenarios**:

1. **Given** a workload configured with `track-tag: "stable"` in YAML, **When** viewed on the Web UI dashboard, **Then** the workload card displays a read-only `Tracked Channel: stable` pill badge (with no editable input).
2. **Given** an existing SQLite database created before this feature, **When** `db.Init` runs, **Then** the `workloads` table is migrated non-destructively to add the `track_tag` column without losing existing rows.
3. **Given** a workload with an update detected on a tracked channel, **When** Apprise sends a notification, **Then** the message includes the target `NewImage` and the `Tracked Channel`.

---

### Edge Cases

- **Missing or invalid `track-tag`**: If `FetchDigest` fails on the configured `track-tag` (e.g. HTTP 404 typo in YAML), the check MUST return an explicit error rather than silently falling back to SemVer tag listing.
- **OCI index containing Buildx attestation manifests**: Multi-arch OCI indexes often include `unknown/unknown` or `vnd.docker.reference.type=attestation-manifest` entries; label lookup MUST skip attestation manifests and prefer a real OS/arch manifest (e.g. `linux/amd64` or `linux/arm64`).
- **Missing `org.opencontainers.image.version` label**: If the tracked tag lacks an OCI version label or the label tag does not verify against the tracked digest, the monitor MUST fall back to scanning compatible higher tags via `ListTags` for a digest match (bounded to avoid excessive API calls), and as a last resort propose `repo:<track-tag>`.
- **Non-version floating current tags (`latest`, `beta`, `edge`)**: Without `track-tag`, non-versionable current tags MUST skip `ListTags` and go directly to same-tag digest comparison.
- **Registry URL with port (`localhost:5000/repo:tag`) or `@sha256:...` digest**: Image repo/tag parsing MUST not misinterpret a registry port colon as the tag separator.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `registry.ParseVersion` MUST classify tags into disjoint schemes (`SchemeSemVer`, `SchemeCalVer`, `SchemeCompactDate`, `SchemeSingleMajor`) and MUST reject malformed or unparseable tags (such as arbitrary CI build integers or concatenated prerelease suffixes like `2026.11.0b1`).
- **FR-002**: `registry.ParseVersion` MUST distinguish prerelease tokens (`alpha`, `beta`, `rc`, `dev`, `nightly`, `next`, `canary`, `preview`, `pre`, `snapshot`, `edge`, `test`, `exp`, `experimental`, `unstable`) from non-prerelease image flavors/variants (`alpine`, `debian`, `legacy`, `r0-ls`, etc.) and extract LinuxServer `-ls<N>` build revisions.
- **FR-003**: `registry.FindLatestMatchingTag` MUST only consider candidate tags whose scheme, numeric segment count, `v`-prefix presence, normalized flavor, and revision presence match the current tag, and MUST exclude prereleases unless the current tag is already a prerelease.
- **FR-004**: `registry.Client` MUST provide `FetchImageLabels` to retrieve OCI image config labels (handling 401 Bearer token challenges, multi-arch indexes, attestation manifest filtering, and cross-host blob redirects cleanly).
- **FR-005**: Kubernetes workload discovery MUST read optional explicit channel tracking from workload annotations (`container-updater.<container>.track-tag` taking precedence over `container-updater.track-tag`, checking `metadata.annotations` with fallback to `spec.template.metadata.annotations`).
- **FR-006**: Docker Compose workload discovery MUST read optional explicit channel tracking from container label `container-updater.track-tag`.
- **FR-007**: The `workloads` SQLite table and `db.Workload` struct MUST store an optional `track_tag` field (`track_tag` in JSON, `omitempty`), with an automatic idempotent migration in `db.Init` for existing databases.
- **FR-008**: When `Workload.TrackTag` is set (and differs from the current tag), `monitor.CheckWorkloadUpdate` MUST compare against the tracked channel digest, resolve the pinned target tag via `org.opencontainers.image.version` (with `v`-prefix adaptation and digest verification) or compatible-tag digest fallback, guard against version downgrades, and only notify/propose updates when the tracked channel represents a newer or updated release.
- **FR-009**: `MonitorScheduler` MUST include `NewImage` and `TrackTag` in Apprise update notifications and notify again if `NewImage` changes while `update_status` is already `update_available`.
- **FR-010**: The Web UI `Workload` interface and `WorkloadItem` component MUST render `track_tag` as a read-only `Tracked Channel` badge when configured and extract tags safely even when image references include registry ports.
- **FR-011**: `README.md` MUST document the schema-aware matching behavior and `container-updater.track-tag` YAML annotations/labels (including multi-container Kubernetes overrides).

### Key Entities

- **Version (`registry.Version`)**: Structured representation of a container tag capturing `Scheme`, `HasVPrefix`, segment `Parts`, numeric components (`Major`, `Minor`, `Patch`, `Build`), `IsPre`, normalized `Flavor`, `RawSuffix`, and LinuxServer `Revision`.
- **Workload (`db.Workload`)**: Extended with optional `TrackTag` (`*string`, serialized as `track_tag`) derived read-only from Kubernetes annotations or Docker labels.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Regression tests verify 0 false-positive tag suggestions across all real-world edge cases (`luligu/matterbridge:3.11.4`, `n8nio/n8n:2.48.7`, `eclipse-mosquitto:2.1.2-alpine`, `caronc/apprise:2.1`, `victoriametrics/victoria-metrics:v1.109.1`, `home-assistant:2026.11.0`, `linuxserver/wireguard:1.0.20260223-r0-ls117`).
- **SC-002**: When `track-tag` digest matches the running tag digest, zero `/v2/*/tags/list` requests are issued during `CheckWorkloadUpdate`.
- **SC-003**: All backend quality gates (`go vet ./...`, `go test -race -count=1 ./...`, `CGO_ENABLED=0 GOOS=linux go build`) and frontend quality gates (`pnpm lint`, `npx tsc --noEmit`, `pnpm build`) pass with zero errors.

## Assumptions

- Tracked channel tags (`stable`, `latest`, etc.) are explicitly opted into per workload via YAML annotations or labels; default mode remains schema-aware version tag tracking.
- Automated update execution (`auto-update`) remains out of scope; the tool notifies and lets the operator trigger updates manually.

## Out of Scope / Future Enhancements

- **Option B — Per-workload regex filters & `update-scope` policies** (`patch` / `minor` / `major` / `digest`), targeted especially at future auto-update guardrails: tracked in GitHub issue [#36](https://github.com/giacomoRanieri/container-updater/issues/36).
- **Option C — GitHub Releases (`/releases/latest`) as an alternative update source**: tracked in GitHub issue [#37](https://github.com/giacomoRanieri/container-updater/issues/37).
