# Research Notes: Container Updater

This document outlines the architectural research, technology selections, and execution patterns chosen for the `container-updater` application.

---

## 1. Docker Container Recreation (Compose Context)

* **Decision**: Use the official Docker Go SDK (`github.com/docker/docker/client`) to inspect, pull, stop, and recreate containers.
* **Rationale**:
  To update a container, the application needs to:
  1. Inspect the running container to capture its exact configurations (ports, volumes, environment variables, networks, restart policy, labels).
  2. Pull the latest image for the specified tag using `cli.ImagePull`.
  3. Stop the existing container using `cli.ContainerStop` (with a timeout).
  4. Rename the old container to a temporary name (e.g., `container-name-old`) to release the host-name and port bindings.
  5. Create a new container using `cli.ContainerCreate` with the same name, configuration, and the newly pulled image digest.
  6. Start the new container using `cli.ContainerStart`.
  7. Remove the old container using `cli.ContainerRemove` once the new one starts successfully.
  If the new container fails to start, the system rolls back by stopping it, removing it, and renaming/restarting the old container.
* **Alternatives Considered**:
  * *Invoking docker-compose CLI*: Hard to manage programmatically inside a docker container, requires mounting Docker CLI and Compose binary. Using the Go Docker client is cleaner and lighter.

---

## 2. Kubernetes Workload Update (k3s/Kubernetes Context)

* **Decision**: Use Kubernetes client-go (`k8s.io/client-go`) to update Deployments/StatefulSets image tags.
* **Rationale**:
  In Kubernetes, updating a container is performed by modifying the image tag/digest on the workload definition (e.g., Deployment spec). Kubernetes then handles the rolling restart (recreating Pods) natively.
  1. Authenticate using `rest.InClusterConfig()` (when running inside k8s) or fallback to `clientcmd.BuildConfigFromFlags` (outside k8s).
  2. Query the workload (e.g., `Deployments.Get`).
  3. Update the `spec.template.spec.containers[i].image` field with the new tag/digest.
  4. Call `Deployments.Update` or patch the deployment.
  5. Let the Kubernetes Control Plane handle rolling updates, health checks, and automatic rollbacks if configured.
* **Alternatives Considered**:
  * *Restarting Pods directly*: Does not update the underlying deployment manifest, leading to reconciliation conflicts. Modifying the Deployment resource is the correct Kubernetes pattern.

---

## 3. Apprise Integration

* **Decision**: Invoke Apprise via an external **Apprise API** service container (HTTP POST webhook requests) or fallback to wrapping the **Apprise CLI** if installed locally.
* **Rationale**:
  Apprise is a Python library. Since our backend is in Go, we cannot import it directly.
  * Standard deployment pattern: Run the official `caronc/apprise-api` container alongside the daemon. The Go backend sends an HTTP POST request to `http://apprise-api:8000/notify` with the Apprise configuration URLs (e.g., `tgram://...`, `https://...` for Home Assistant) and notification content.
  * This keeps the Go backend CGO-free, lightweight, and delegates notification endpoints configuration entirely to the robust Apprise ecosystem.
* **Alternatives Considered**:
  * *Writing custom Telegram/HA integration in Go*: Re-inventing the wheel, lacks the 80+ notification integrations that Apprise offers out of the box.
  * *Wrapping apprise CLI*: Requires installing Python and Apprise inside our Go container, increasing image size.

---

## 4. SQLite Pure Go Driver

* **Decision**: Use `modernc.org/sqlite` (integrated via GORM with `github.com/glebarez/sqlite` or directly via standard library `database/sql`).
* **Rationale**:
  Standard Go SQLite drivers (like `github.com/mattn/go-sqlite3`) rely on CGO, which requires a C compiler for compiling and makes cross-compilation (e.g., building a Linux ARM64 container from a MacOS AMD64 system) difficult.
  `modernc.org/sqlite` is a pure-Go transpilation of SQLite that runs without CGO, making the compilation workflow simple and highly portable.
* **Alternatives Considered**:
  * *mattn/go-sqlite3*: Rejected due to CGO compilation requirements.

---

## 5. OAuth2 / OIDC Integration

* **Decision**: Use standard OIDC authentication in the Go Backend (using `github.com/coreos/go-oidc/v3/oidc`) with state validation, and Next.js handling redirects and storing JWTs.
* **Rationale**:
  The user configures an OIDC provider (Authelia, Authentik, Keycloak) with client ID, client secret, and issuer URL.
  1. Frontend redirects unauthenticated users to `/api/auth/login` on the Go backend, which redirects to the OIDC provider.
  2. After login, OIDC provider redirects to Go backend callback endpoint `/api/auth/callback` with an auth code.
  3. Go backend exchanges code for tokens, validates the ID token, and issues a secure HTTP-only cookie containing a JWT session.
  4. Next.js UI queries `/api/auth/session` to check login status and user info.
* **Alternatives Considered**:
  * *NextAuth.js on Frontend*: Feasible, but securing the REST API of the Go backend requires the backend to validate tokens anyway. Centralizing token validation on the Go backend is more secure.
