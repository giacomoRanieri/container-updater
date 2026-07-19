# Container Updater 🚀

Container Updater is a powerful automated utility to monitor running container workloads (Docker Compose & Kubernetes), check for tag/digest updates in remote registries, execute safe automated updates, persist log audits, and dispatch Markdown alerts via Apprise (Telegram & Home Assistant) including release notes fetched from GitHub.

---

## 🛠️ Architecture

* **Backend**: Go daemon + Chi REST API + WebSocket Hub. Persistent storage via CGO-free pure-Go SQLite driver.
* **Frontend**: Next.js (React) + TypeScript console styled with premium, responsive glassmorphic Vanilla CSS.
* **Notifications**: Integrates with Apprise API to support Home Assistant, Telegram, and 80+ notification services.

---

## 🚀 Quickstart with Docker Compose

### 1. Configure docker-compose.yml
Use the preconfigured `docker-compose.yml` in the root folder. Ensure that the Docker socket is mounted to allow container management:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
  - ./data:/app/data
```

### 2. Build and Run the Stack
Run the following command to build and launch all services:

```bash
docker compose up -d --build
```

Access the console at:
* **Frontend Dashboard**: `http://localhost:3000`
* **Backend API**: `http://localhost:8080`
* **Apprise API**: `http://localhost:8000`

---

## 📦 Monitoring Workloads

To allow Container Updater to monitor a service, add the following labels or annotations:

### Docker Compose
Add the label `container-updater.enable=true` to your service:
```yaml
services:
  nginx-frontend:
    image: nginx:1.25.0
    labels:
      - "container-updater.enable=true"
```

### Kubernetes
Add the annotation `container-updater.enable: "true"` to your Deployment, StatefulSet, or DaemonSet metadata:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    container-updater.enable: "true"
spec: ...
```

---

## ⚙️ Environment Variables

### Backend Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port for the backend API |
| `DATABASE_PATH` | `/app/data/db.sqlite` | SQLite database persistence path |
| `LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`) |
| `CRON_SCHEDULE` | `*/15 * * * *` | Scheduled checks interval (cron syntax) |
| `GITHUB_TOKEN` | *None* | Optional GitHub API token to prevent rate limits during changelog lookup |
| `APPRISE_API_URL` | `http://localhost:8000` | Connection string for Apprise notification API |
| `OIDC_ENABLED` | `false` | Enable OIDC single sign-on authentication |
| `COMPOSE_PATH_MAP` | *None* | Maps container host paths to local mounts |

### GitOps Configuration

| Variable | Default | Description |
|---|---|---|
| `GITOPS_ENABLED` | `false` | Enable automated Git push on Kubernetes manifest updates |
| `GITOPS_REPO_URL` | *None* | Remote Git repository SSH/HTTPS URL |
| `GITOPS_BRANCH` | `main` | Target repository branch |
| `GITOPS_SSH_KEY_PATH`| *None* | SSH Private key file path for authentication |
| `GITOPS_USERNAME` | *None* | HTTP Basic Auth username |
| `GITOPS_PASSWORD` | *None* | HTTP Basic Auth password/PAT token |

---

## 🖥️ Local Development

### Prerequisites
* Go 1.22+
* Node.js 20+ & pnpm

### Running the Backend
```bash
cd backend
go run cmd/updater/main.go
```

### Running the Frontend
```bash
cd frontend
pnpm install
pnpm dev
```
Open `http://localhost:3000` in your browser. Mock authentication will automatically sign you in as `admin` if OIDC is disabled.
