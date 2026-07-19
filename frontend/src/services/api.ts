const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const WS_URL = BACKEND_URL.replace(/^http/, "ws") + "/api/ws";

export interface Workload {
  id: string;
  name: string;
  namespace_project: string;
  orchestrator_type: "compose" | "kubernetes";
  current_image: string;
  current_digest: string;
  new_image?: string;
  new_digest?: string;
  update_status: "up_to_date" | "update_available" | "updating" | "failed";
  last_checked_at?: string;
  last_updated_at?: string;
}

export interface AuditLogItem {
  id: string;
  workload_id: string;
  workload_name: string;
  status: "pending" | "in_progress" | "completed" | "failed" | "rolled_back";
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface AppStats {
  total_workloads: number;
  updates_available: number;
  successful_updates: number;
  last_check_at?: string;
}

export interface NotificationService {
  id: string;
  name: string;
  apprise_url: string;
  is_enabled: boolean;
  created_at?: string;
}

// Fetch helper with standard headers and credentials
async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = `${BACKEND_URL}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      "Accept": "application/json",
      "Content-Type": "application/json",
      ...options.headers,
    },
    credentials: "include",
  });

  if (!response.ok) {
    if (response.status === 401) {
      // Prompt OIDC redirect
      if (typeof window !== "undefined") {
        window.location.href = `${BACKEND_URL}/api/auth/login`;
      }
    }
    throw new Error(`API error ${response.status}: ${response.statusText}`);
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

export async function fetchWorkloads(): Promise<Workload[]> {
  return apiRequest<Workload[]>("/api/workloads");
}

export async function fetchAuditLogs(): Promise<AuditLogItem[]> {
  return apiRequest<AuditLogItem[]>("/api/audit-logs");
}

export async function fetchStats(): Promise<AppStats> {
  return apiRequest<AppStats>("/api/stats");
}

export async function fetchNotifications(): Promise<NotificationService[]> {
  return apiRequest<NotificationService[]>("/api/notifications");
}

export async function saveNotification(service: Omit<NotificationService, "id"> & { id?: string }): Promise<NotificationService> {
  return apiRequest<NotificationService>("/api/notifications", {
    method: "POST",
    body: JSON.stringify(service),
  });
}

export async function triggerWorkloadUpdate(id: string): Promise<{ job_id: string; status: string; message: string }> {
  return apiRequest<{ job_id: string; status: string; message: string }>(`/api/workloads/${id}/update`, {
    method: "POST",
  });
}

// WebSocket real-time subscription
export function connectWebSocket(
  onEvent: (event: string, data: any) => void,
  onClose?: () => void
): () => void {
  let ws: WebSocket | null = null;
  let reconnectTimer: NodeJS.Timeout | null = null;
  let isClosed = false;

  function connect() {
    if (isClosed) return;

    logger("Attempting WebSocket connection to " + WS_URL);
    ws = new WebSocket(WS_URL);

    ws.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (payload && payload.event) {
          onEvent(payload.event, payload.data);
        }
      } catch (err) {
        console.error("Failed to parse WebSocket message:", err);
      }
    };

    ws.onclose = () => {
      logger("WebSocket connection closed.");
      if (onClose) onClose();
      // Retry connection after 5 seconds if not explicitly closed by client
      if (!isClosed) {
        reconnectTimer = setTimeout(connect, 5000);
      }
    };

    ws.onerror = (err) => {
      console.error("WebSocket connection error:", err);
    };
  }

  connect();

  // Return un-subscribe cleanup function
  return () => {
    isClosed = true;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    if (ws) ws.close();
  };
}

function logger(msg: string) {
  if (process.env.NODE_ENV !== "production") {
    console.log(`[WebSocket] ${msg}`);
  }
}
