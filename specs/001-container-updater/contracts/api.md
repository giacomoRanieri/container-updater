# API Contracts: Container Updater

This document defines the HTTP REST and WebSocket contracts for communication between the React/Next.js frontend and the Go backend.

---

## 1. Authentication (OIDC/OAuth2)

All API endpoints (except `/api/auth/*`) require a valid session JWT cookie named `session_token`.

### 1.1. Login Redirect
* **Endpoint**: `GET /api/auth/login`
* **Response**: `302 Redirect` to the configured OIDC provider authorization URL.

### 1.2. Callback
* **Endpoint**: `GET /api/auth/callback`
* **Params**: `code` (auth code), `state` (OIDC state token)
* **Response**: `302 Redirect` to `/` (frontend home) on success, setting the `session_token` HTTP-only cookie.

### 1.3. Session Info
* **Endpoint**: `GET /api/auth/session`
* **Response**: `200 OK`
  ```json
  {
    "authenticated": true,
    "user": {
      "id": "usr-12345",
      "username": "admin",
      "email": "admin@example.com"
    }
  }
  ```

---

## 2. Workloads Management

### 2.1. List Workloads
* **Endpoint**: `GET /api/workloads`
* **Response**: `200 OK`
  ```json
  [
    {
      "id": "docker-compose-nginx",
      "name": "nginx",
      "namespace_project": "prod-stack",
      "orchestrator_type": "compose",
      "current_image": "nginx:1.25.0",
      "current_digest": "sha256:456c...",
      "new_image": "nginx:1.25.1",
      "new_digest": "sha256:789d...",
      "update_status": "update_available",
      "last_checked_at": "2026-07-18T18:00:00Z",
      "last_updated_at": "2026-07-18T15:30:00Z"
    }
  ]
  ```

### 2.2. Trigger Workload Update
* **Endpoint**: `POST /api/workloads/{id}/update`
* **Response**: `202 Accepted`
  ```json
  {
    "job_id": "job-8888-9999",
    "status": "pending",
    "message": "Update job scheduled successfully."
  }
  ```

---

## 3. Configuration & Notifications

### 3.1. List Notification Services
* **Endpoint**: `GET /api/notifications`
* **Response**: `200 OK`
  ```json
  [
    {
      "id": "notify-telegram",
      "name": "Telegram Chat",
      "apprise_url": "tgram://12345:ABC/6789",
      "is_enabled": true
    }
  ]
  ```

### 3.2. Save Notification Service
* **Endpoint**: `POST /api/notifications`
* **Body**:
  ```json
  {
    "name": "Telegram Chat",
    "apprise_url": "tgram://12345:ABC/6789",
    "is_enabled": true
  }
  ```
* **Response**: `201 Created`

---

## 4. WebSockets: Real-time Workload Updates

Clients can connect to the WebSocket endpoint to receive push notifications when workload statuses change or when an update job state changes.

* **Endpoint**: `GET /api/ws`
* **Protocol**: WS / WSS
* **Messages emitted by Server**:
  ```json
  {
    "event": "workload_updated",
    "data": {
      "id": "docker-compose-nginx",
      "update_status": "updating"
    }
  }
  ```
  ```json
  {
    "event": "job_status",
    "data": {
      "job_id": "job-8888-9999",
      "status": "in_progress",
      "percent_complete": 50,
      "message": "Pulling new image: nginx:1.25.1"
    }
  }
  ```
