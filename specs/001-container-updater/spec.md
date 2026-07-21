# Feature Specification: Container Updater

**Feature Branch**: `001-container-updater`

**Created**: 2026-07-18

**Status**: Draft

**Input**: User description: "L'obiettivo di container-updater è avere un applicazione che monitora ad intervalli regolari lo stato dei miei container per capire se esiste una nuova versione da scaricare. Manderà notifiche tramite apprise ad home assistant o telegram. Avrà a disposizione una ui web per permettere di lanciare gli aggiornamenti dei container."

## Clarifications

### Session 2026-07-18

- **Q1: Which orchestrators/runtimes should be supported for container updates?** → **A**: Kubernetes (including k3s) and Docker Compose.
- **Q2: How should access to the Web UI be secured?** → **A**: Multi-user OAuth2/OIDC authentication, configurable with providers like Authelia, Authentik, or others.
- **Q3: How should configuration of registries, schedules, and notifications be managed?** → **A**: Via configuration files (YAML/JSON) and dynamically via the Web UI (persisted in a database).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automatic Update Monitoring & Notifications (Priority: P1)

As a system administrator, I want the system to periodically check if new versions of the Docker images running in my containers are available on the registries, and notify me via Apprise (supporting Home Assistant and Telegram), so that I am always aware of available updates.

**Why this priority**: Core value of the application is identifying when updates are available and notifying the user.

**Independent Test**: Can be tested by running the application against a test container running an outdated image tag, verifying that the check triggers at the configured interval, detects the update, and successfully sends a notification payload via Apprise.

**Acceptance Scenarios**:

1. **Given** a container running image `nginx:1.25.0` and a registry containing `nginx:1.25.1` (or a newer digest for the same tag), **When** the scheduled check executes, **Then** the system detects a newer version is available.
2. **Given** a container with an update detected, **When** the check completes, **Then** a notification is dispatched via Apprise containing the container name, current image version/digest, new image version/digest, and a snippet of the latest GitHub releases changelog (if the image has a linked GitHub source).
3. **Given** a container running the latest version available on the registry, **When** the scheduled check executes, **Then** no update is detected and no notification is sent.

---

### User Story 2 - Web UI Dashboard & Authentication (Priority: P2)

As a system administrator, I want to open a Web UI dashboard securely using my OAuth2 provider (like Authelia or Authentik) to view the list of monitored containers/workloads, their current status, and whether an update is available for each, so that I have a clear overview of my environment.

**Why this priority**: High value for visibility and control, providing the interface for manual update actions.

**Independent Test**: Can be fully tested by launching the Web UI, attempting to access it without authentication, verifying redirection to the OAuth2 provider, logging in, and verifying that it correctly lists running containers with their update statuses.

**Acceptance Scenarios**:

1. **Given** the backend has scanned the containers, **When** the authenticated administrator loads the Web UI, **Then** a clean dashboard is displayed listing all monitored containers with their names, namespace/project, status (running/stopped), current image tags, and update status ("Up to date" or "Update available").
2. **Given** a container has an update available, **When** viewed on the dashboard, **Then** an "Update" button is enabled next to that container.
3. **Given** the system is configured with an OAuth2 provider, **When** an unauthenticated user visits the Web UI, **Then** they are redirected to the identity provider login screen.
4. **Given** update jobs have been executed in the past, **When** the administrator views the Audit Log section, **Then** a table is displayed showing a list of past jobs with container names, status (success/failed), execution times, and any associated error details.

---

### User Story 3 - Manual Update Execution from Web UI (Priority: P1)

As a system administrator, I want to click an "Update" button on the Web UI for a specific container to trigger its update immediately, so that I can control when my services go through a restart/update.

**Why this priority**: Crucial for the system's active updating capability (distinguishing it from a purely read-only monitoring tool).

**Independent Test**: Can be tested by clicking the update button on the Web UI, verifying that the backend stops the container/workload, pulls the new image, and recreates the container (or updates the deployment) with the original configurations, showing a success message in the UI once completed.

**Acceptance Scenarios**:

1. **Given** a Docker Compose service has an update available, **When** the administrator clicks the "Update" button, **Then** the backend updates the image tag in the corresponding `docker-compose.yml` file on disk, runs `docker compose up -d` for that file, and verifies the service restarted successfully with the new image.
2. **Given** a Kubernetes Deployment has an update available, **When** the administrator clicks the "Update" button, **Then** the backend updates the image tag/digest in the local Kubernetes manifest YAML on disk, applies it, and verifies the deployment rolling update is initiated.
3. **Given** a container update is in progress, **When** another update request is sent for the same container, **Then** the system rejects it and indicates that an update is already in progress.

---

### User Story 4 - GitOps Git Integration (Priority: P3)

As a system administrator, I want the system to commit and push updated YAML files to a Git repository when an update occurs, so that my Git-based configuration stays in sync with my running workloads.

**Why this priority**: Crucial for users employing GitOps workflows to maintain Git as the single source of truth.

**Independent Test**: Can be tested by triggering an update, verifying that manifest changes are written to disk, and confirming a new commit is pushed to the target Git repository containing the image update diff.

**Acceptance Scenarios**:

1. **Given** Git integration is enabled, **When** a Compose or Kubernetes YAML manifest is updated, **Then** the backend commits the changed file with a message (e.g., "chore: update dummy-nginx to image:tag") and pushes it to the configured remote repository.

---

### User Story 5 - Repository Version Management & CI/CD Release Automation (Priority: P2)

As a project maintainer and contributor, I want the repository to enforce Conventional Commits, automated Semantic Versioning, protected branches, and CI/CD pipelines for Beta and Production releases, so that external PRs can be safely reviewed/approved by maintainers and published to GitHub Releases and GHCR without manual intervention.

**Why this priority**: Essential for software quality, security governance, automated changelogs, and reproducible multi-architecture releases.

**Independent Test**: Can be tested by submitting a PR with Conventional Commit formatting, verifying CI status checks and maintainer approval enforcement, merging to `develop` to verify automated Beta image publishing, and merging a Release-Please PR to `master` to confirm GitHub Release creation and tagged GHCR Docker image deployment.

### User Story 5 - Repository Version Management & CI/CD Automation (Priority: P2)

As a maintainer or external contributor, I want automated conventional commit checks, automated SemVer versioning and CHANGELOG generation via Release-Please, multi-architecture Docker image builds for Beta (`develop`) and GA (`master`), CODEOWNERS branch protection, PR preview container lifecycles, and environment approval gates, so that the project maintains strict software quality standards, automated release channels, and secure governance.

#### Acceptance Criteria
- **AC-5.1**: Automated CI (`commitlint`) validates Conventional Commits for all commit headers and PR titles.
- **AC-5.2**: Merges to `develop` trigger Release-Please to update `CHANGELOG.md` and manage SemVer pre-releases (`vX.Y.Z-beta.N`), and publish multi-arch Docker images (`:beta`, `:beta-<sha>`) to GHCR.
- **AC-5.3**: Merges of Release PRs to `master` automatically generate official GitHub Releases, publish source archives, and tag production Docker images (`:latest`, `:vX.Y.Z`, `:vX.Y`, `:vX`) on GHCR.
- **AC-5.4**: Branch protection rules on `master` and `develop` block direct pushes, require passing status checks, and mandate at least 1 maintainer approval (`.github/CODEOWNERS`).
- **AC-5.5**: Pull Requests generate temporary preview Docker images (`:pr-<N>`) on GHCR that are automatically deleted upon PR closure or merge via `pr-cleanup.yml`.
- **AC-5.6**: Workflow deployment jobs incorporate GitHub Environment Approval Gates (`environment: beta-build`), requiring maintainer review before image publishing.

### User Story 6 - Docker Compose GitOps & File-Based Kubernetes Updates (Priority: P2)

As a DevOps engineer or system administrator, I want `container-updater` to support GitOps repository synchronization for Docker Compose stacks AND local file-based manifest updates for Kubernetes workloads, so that Compose configurations can be declaratively version-controlled in Git, and Kubernetes manifests on local disk can be updated directly without requiring cluster API write permissions.

#### Acceptance Criteria
- **AC-6.1**: When `COMPOSE_GITOPS_ENABLED=true`, `container-updater` clones a configured Git repository containing `docker-compose.yml` files, updates `image:` tags upon new release detection, and commits & pushes changes back to the remote Git repository.
- **AC-6.2**: When `K8S_FILE_BASED_ENABLED=true`, `container-updater` scans a configured local directory (`K8S_MANIFEST_DIR`) for Kubernetes Deployment, StatefulSet, and DaemonSet YAML files.
- **AC-6.3**: File-based Kubernetes updates parse local YAML manifests on disk, update the target `spec.template.spec.containers[].image` tag, and preserve original YAML formatting and indentation.
- **AC-6.4**: File-based Kubernetes updates optionally support executing `kubectl apply -f <manifest>` or triggering local GitOps sync.
- **AC-6.5**: The Web UI and API expose configuration fields for Compose GitOps repository settings (`COMPOSE_GITOPS_REPO_URL`, `COMPOSE_GITOPS_BRANCH`) and Kubernetes File-Based manifest paths (`K8S_MANIFEST_DIR`, `K8S_MANIFEST_PATH_MAP`).
- **AC-6.6**: When `K8S_AUTO_APPLY=true`, `container-updater` automatically executes `kubectl apply -f <manifest>` after modifying the local Kubernetes YAML file on disk.
- **AC-6.7**: For GitOps controllers (ArgoCD/FluxCD), `container-updater` supports pushing modified manifests to a Git repository OR dispatching an HTTP Sync Webhook (`K8S_SYNC_WEBHOOK_URL`).

---

### Edge Cases

- **Registry Rate Limiting / Timeout**: How does the system handle registry authentication errors or Docker Hub rate limits (especially for anonymous pulls)?
- **Failed Container Startup After Update**: If the new image fails to start or crashes immediately, does the system perform an automatic rollback to the previous version?
- **Network Disconnection**: If the network is down during a check, the system must log the error, skip notifications, and retry during the next scheduled run without crashing.
- **Concurrent Updates**: If multiple containers belong to the same Docker Compose stack or depend on each other, updating them in isolation might break dependencies.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST monitor container images at a user-defined cron interval or duration-based schedule.
- **FR-002**: The system MUST fetch remote registry metadata (digests or tags) to compare them with the locally running container image digests.
- **FR-003**: The system MUST support Apprise integration to send notifications to Telegram and Home Assistant.
- **FR-004**: The system MUST expose a REST API to query container status and trigger updates.
- **FR-005**: The system MUST expose a Web UI that displays the monitored containers and allows triggering updates.
- **FR-006**: The system MUST support updating workloads by modifying configuration files (Docker Compose YAML files and Kubernetes manifests) on disk.
- **FR-007**: Access to the Web UI MUST be secured via multi-user OAuth2/OIDC authentication, supporting configurable providers such as Authelia and Authentik.
- **FR-008**: The configuration of registries, schedules, and notifications MUST be definable via a configuration file (YAML/JSON) and manageable via a settings page in the Web UI (persisted in a database).
- **FR-009**: The container-updater application itself MUST be deployable as a Docker container, supporting multi-architecture builds (linux/amd64 and linux/arm64).
- **FR-010**: The system MUST support GitOps integration to commit and push changed configuration files to a configured Git repository.
- **FR-011**: The system MUST only scan workloads that have the label `container-updater.enable=true` (Docker) or annotation `container-updater.enable: "true"` (Kubernetes) enabled.
- **FR-012**: The system MUST support storing and using registry authentication credentials to check private registries.
- **FR-013**: The system MUST log all update actions and maintain an audit log / job history visible in the Web UI.
- **FR-014**: The system MUST extract the GitHub repository source from image labels (e.g. `org.opencontainers.image.source`) and fetch the release notes / changelog from the GitHub API to enrich notification messages.
- **FR-015**: The project MUST include GitHub Actions workflows to natively build and publish multi-architecture Docker images (linux/amd64 and linux/arm64) to GitHub Container Registry (GHCR) using matrix native runners (without QEMU emulation).
- **FR-016**: The project MUST enforce Conventional Commits specification for all commit messages and PR titles via CI automated checks (`commitlint`).
- **FR-017**: The project MUST use Release-Please (or equivalent SemVer release automation) to automatically calculate Semantic Versioning increments (Major, Minor, Patch) from Conventional Commits, update `CHANGELOG.md`, and generate Release PRs.
- **FR-018**: The repository MUST implement Branch Protection rules on `master` and `develop` branches: blocking direct pushes, requiring at least 1 approval from a designated Maintainer (`.github/CODEOWNERS`), and requiring passing CI status checks before merging.
- **FR-019**: Merges to `develop` MUST automatically trigger CI pipelines to build and push Beta multi-arch Docker images to GHCR (tagged `:beta` and `:vX.Y.Z-beta.N`) and create GitHub Pre-releases.
- **FR-020**: Merges of Release PRs to `master` MUST automatically publish official GitHub Releases with source code archives, binary checksums, and publish tagged Docker images to GHCR (`:latest`, `:vX.Y.Z`, `:vX.Y`, `:vX`).
- **FR-021**: Pull Requests MUST produce temporary PR Preview Docker images (`:pr-<N>`) on GHCR that exist only for the lifespan of the PR, and MUST be automatically deleted upon PR closure or merge via an automated cleanup workflow (`pr-cleanup.yml`).
- **FR-022**: CI/CD pipelines MUST integrate GitHub Environment Approval Gates (`environment: beta-build`), requiring designated maintainer sign-off before publishing Docker images. The workflow MUST gate image publish steps behind this environment so beta and preview artifacts are not pushed without approval.
- **FR-023**: CI/CD workflows MUST implement concurrency cancellation rules (`concurrency: cancel-in-progress: true`) to automatically abort redundant older builds when new commits are pushed. The workflow MUST ensure newer runs supersede in-flight builds for the same branch or PR.
- **FR-024**: The system MUST support GitOps repository synchronization for Docker Compose stacks (`COMPOSE_GITOPS_ENABLED=true`), automatically committing and pushing updated `docker-compose.yml` `image:` tags to a remote Git repository.
- **FR-025**: The system MUST support local file-based manifest updates for Kubernetes workloads (`K8S_FILE_BASED_ENABLED=true`), scanning a configured local directory (`K8S_MANIFEST_DIR`) for Kubernetes Deployment, StatefulSet, and DaemonSet YAML files.
- **FR-026**: When file-based Kubernetes updating is active, the system MUST parse local Kubernetes YAML manifests on disk, update the target `spec.template.spec.containers[].image` tag value, and preserve original YAML formatting and indentation.
- **FR-027**: The system MUST support mapping local Kubernetes manifest directories to host mounts (`K8S_MANIFEST_PATH_MAP`), allowing containerized `container-updater` deployments to locate and modify host Kubernetes YAML files.
- **FR-028**: The Web UI settings page and REST API MUST expose configuration controls for Docker Compose GitOps (`COMPOSE_GITOPS_REPO_URL`, `COMPOSE_GITOPS_BRANCH`) and Kubernetes File-Based manifest options (`K8S_MANIFEST_DIR`, `K8S_FILE_BASED_ENABLED`, `K8S_MANIFEST_PATH_MAP`).
- **FR-029**: The system MUST support automatic cluster apply (`K8S_AUTO_APPLY=true`) and external reconciliation webhooks (`K8S_SYNC_WEBHOOK_URL`) upon completing file-based Kubernetes manifest modifications.
- **FR-030**: The frontend application MUST support runtime environment variable injection for backend API URL (`NEXT_PUBLIC_BACKEND_URL`) via `public/env-config.js` and container entrypoint script (`docker-entrypoint.sh`), allowing containerized frontend deployments to connect to dynamic backend URLs.
- **FR-031**: The backend OIDC authentication service MUST support `OIDC_PROVIDER_URL` as an environment variable fallback alias for `OIDC_ISSUER`, maintaining compatibility across Kubernetes ConfigMap templates and environment configurations.
- **FR-032**: The backend authentication module MUST dynamically resolve the frontend redirect target URL (`FRONTEND_URL` / `OIDC_FRONTEND_URL`) after successful OIDC or mock authentication callbacks, using configured environment variables, request `Referer` headers, or `Host` headers, eliminating hardcoded host assumptions.
- **FR-033**: The database layer MUST scan aggregate SQL query timestamp results (e.g. `MAX(last_checked_at)`) safely as generic interface types or strings and parse them using multi-format timestamp parsers (`parseSQLiteTime`), preventing SQLite type scan mismatches during stats calculation (`/api/stats`).
- **FR-034**: The monitor scheduler MUST execute remote registry digest inspection (`CheckWorkloadUpdate`) for all scanned workloads (including Kubernetes workloads) regardless of whether the local Docker daemon client socket is connected or available.
- **FR-035**: The system MUST implement a native HTTP/HTTPS OCI Registry v2 client (`registry.Client`) to query remote image digests directly over HTTP without requiring a running Docker Engine daemon or socket.
- **FR-036**: The Web UI workload card component MUST render status badges (`UP TO DATE`, `UPDATE AVAILABLE`) using responsive flexbox positioning without absolute overlapping, enforce text truncation on long names, and format Kubernetes workload names concisely (`formatWorkloadName`).
- **FR-037**: The system MUST expose a protected `POST /api/scan` API endpoint allowing users to trigger an immediate, on-demand background registry update check across all monitored workloads without waiting for the scheduled cron interval.
- **FR-038**: The monitor service MUST support Semantic Versioning (SemVer) and CalVer tag tracking (`registry.FindLatestMatchingTag`), listing remote registry tags via OCI v2 API (`GET /v2/<repo>/tags/list`) and detecting newer release version tags (e.g. `2026.6.4` -> `2026.6.5`) even when image tags are version-pinned.
- **FR-039**: The OCI registry HTTP client MUST parse `Www-Authenticate` Bearer header parameters using an RFC 7235–compliant quoted-string-aware parser, ensuring the `scope` field is correctly extracted even when it contains colons or commas inside quoted values.
- **FR-040**: The OCI registry HTTP client `ListTags` implementation MUST paginate through all registry tag pages by following RFC 5988 `Link: rel="next"` response headers until all tags are fetched, preventing missed newer tags for repositories with more than 100 releases.

### Key Entities *(include if feature involves data)*

- **Container / Workload**: Represents a running container in Docker Compose or a workload (Deployment, StatefulSet, DaemonSet) in Kubernetes. Attributes include ID/Name, Namespace/Project, Orchestrator Type (Compose vs Kubernetes), Current Image, Current Digest, New Digest (if update available), State (running, stopped, updating).
- **User**: Represents an authenticated user session with identity details from OAuth2.
- **Notification Service**: Represents a configured Apprise notification endpoint (e.g., Telegram chat, Home Assistant webhook).
- **Update Job**: Represents the execution state of a container/workload update (Pending, Pulling, Stopping, Recreating, Verifying, Completed, Failed).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The scheduled check completes scanning 10 containers and checking their registry status in under 30 seconds (assuming a stable network connection).
- **SC-002**: The system correctly identifies image digest mismatches for 100% of scanned containers.
- **SC-003**: Notifications are received on Telegram/Home Assistant within 5 seconds of the check finishing.
- **SC-004**: Manual updates triggered via the Web UI recreate the container or update the deployment with zero loss of original configurations.
- **SC-005**: The Web UI updates container state dynamically or reflects the update progress within 1 second of API state changes.

## Assumptions

- The host running the container-updater has access to the Docker socket (for Compose) and/or Kubernetes API (for K8s clusters).
- An external OAuth2/OIDC provider (such as Authelia or Authentik) is available and configured by the administrator.
- Private registry authentication (if needed) will be supported for common registries (Docker Hub, GitHub Packages, GitLab Container Registry).
- The network is generally available, but intermittent offline states are handled gracefully.
