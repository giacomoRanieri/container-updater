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
2. **Given** a container with an update detected, **When** the check completes, **Then** a notification is dispatched via Apprise containing the container name, current image version/digest, and new image version/digest.
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

---

### User Story 3 - Manual Update Execution from Web UI (Priority: P1)

As a system administrator, I want to click an "Update" button on the Web UI for a specific container to trigger its update immediately, so that I can control when my services go through a restart/update.

**Why this priority**: Crucial for the system's active updating capability (distinguishing it from a purely read-only monitoring tool).

**Independent Test**: Can be tested by clicking the update button on the Web UI, verifying that the backend stops the container/workload, pulls the new image, and recreates the container (or updates the deployment) with the original configurations, showing a success message in the UI once completed.

**Acceptance Scenarios**:

1. **Given** a Docker Compose service has an update available, **When** the administrator clicks the "Update" button, **Then** the backend pulls the new image and restarts/recreates the compose service using Compose commands, preserving original volumes, ports, environment variables, and network configurations.
2. **Given** a Kubernetes Deployment has an update available, **When** the administrator clicks the "Update" button, **Then** the backend updates the deployment image tag/digest to trigger a rolling update in the cluster.
3. **Given** a container update is in progress, **When** another update request is sent for the same container, **Then** the system rejects it and indicates that an update is already in progress.

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
- **FR-006**: The system MUST support updating workloads running in Docker Compose stacks and Kubernetes/k3s clusters.
- **FR-007**: Access to the Web UI MUST be secured via multi-user OAuth2/OIDC authentication, supporting configurable providers such as Authelia and Authentik.
- **FR-008**: The configuration of registries, schedules, and notifications MUST be definable via a configuration file (YAML/JSON) and manageable via a settings page in the Web UI (persisted in a database).

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
