# Quickstart Validation Guide: Container Updater

This guide provides scenarios to validate that the `container-updater` application functions correctly end-to-end.

---

## 1. Prerequisites

1. **Docker environment**: Docker Engine running locally with access to `/var/run/docker.sock`.
2. **K3s/Kubernetes cluster**: A local k3s cluster or kubectl configured to access a cluster.
3. **Apprise API**: Run an Apprise API container:
   ```bash
   docker run -d --name apprise-api -p 8000:8000 caronc/apprise-api
   ```
4. **Mock OIDC Provider**: Configure a mock OIDC provider (or Authentik/Authelia) for OAuth2 authentication flows.

---

## 2. Setup and Execution

To run the full stack for validation:

1. **Build and Run Backend**:
   Ensure a local SQLite database file `db.sqlite` is available.
   ```bash
   cd backend
   go build -o updater cmd/updater/main.go
   ./updater --config config.yaml
   ```

2. **Build and Run Frontend**:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

---

## 3. End-to-End Validation Scenarios

### Scenario A: Automatic Update Detection and Notification (Compose)

* **Goal**: Validate that the backend detects outdated images and sends notifications via Apprise.
* **Steps**:
  1. Start a dummy container running an old image:
     ```bash
     docker run -d --name dummy-nginx --label "container-updater.enable=true" nginx:1.25.0
     ```
  2. Configure Apprise Telegram notification URL in the SQLite database or `config.yaml`.
  3. Wait for the scheduled check interval (or trigger a check manually).
* **Expected Outcome**:
  * The backend detects that a newer version of `nginx` is available.
  * A Telegram message is sent containing:
    > "Update available for dummy-nginx: nginx:1.25.0 -> nginx:1.25.1"
  * In the SQLite `workloads` table, the row for `dummy-nginx` is updated with `new_image` and `update_status='update_available'`.

### Scenario B: Manual Update Trigger from Web UI (Compose)

* **Goal**: Validate that the UI can successfully trigger and execute a container update.
* **Steps**:
  1. Open the Web UI at `http://localhost:3000` and authenticate.
  2. Locate the workload `dummy-nginx` displaying "Update Available".
  3. Click the **Update** button.
* **Expected Outcome**:
  * The UI displays "Updating...".
  * The backend pulls `nginx:1.25.1`.
  * The backend stops and recreates `dummy-nginx` with the new image, keeping original parameters.
  * The UI changes status to "Up to Date" once the update completes.
  * Running `docker inspect dummy-nginx` shows the image is now `nginx:1.25.1` (or latest digest).

### Scenario C: Kubernetes Deployment Update

* **Goal**: Validate K8s deployment image update.
* **Steps**:
  1. Deploy a sample deployment in the cluster:
     ```bash
     kubectl create deployment dummy-k8s --image=nginx:1.25.0
     ```
  2. Run the updater inside the same cluster with appropriate RBAC roles.
  3. Locate `dummy-k8s` in the Web UI dashboard and click **Update**.
* **Expected Outcome**:
  * The backend updates the image reference to `nginx:1.25.1` in the Deployment resource.
  * A rolling update is initiated in Kubernetes.
  * `kubectl get deployment dummy-k8s -o jsonpath='{.spec.template.spec.containers[0].image}'` returns `nginx:1.25.1`.
