import React from "react";
import { AuditLogItem } from "../services/api";

interface AuditLogProps {
  logs: AuditLogItem[];
  onRefresh: () => void;
  isLoading: boolean;
}

export default function AuditLog({ logs, onRefresh, isLoading }: AuditLogProps) {
  const getDuration = (start: string, end: string) => {
    const sTime = new Date(start).getTime();
    const eTime = new Date(end).getTime();
    const diff = eTime - sTime;
    
    if (isNaN(diff) || diff <= 0) {
      return "< 1s";
    }

    const secs = Math.floor(diff / 1000);
    if (secs < 60) {
      return `${secs}s`;
    }
    const mins = Math.floor(secs / 60);
    return `${mins}m ${secs % 60}s`;
  };

  const getStatusClass = (status: string) => {
    switch (status) {
      case "completed":
        return "status-up_to_date";
      case "failed":
        return "status-failed";
      case "in_progress":
        return "status-updating";
      case "rolled_back":
        return "status-warning";
      default:
        return "status-info";
    }
  };

  return (
    <div className="audit-log-section">
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1.5rem" }}>
        <h2 style={{ fontSize: "1.3rem" }}>Update Audit Log & History</h2>
        
        <button
          className="btn btn-secondary"
          onClick={onRefresh}
          disabled={isLoading}
        >
          <svg
            className={isLoading ? "spin" : ""}
            width="16"
            height="16"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            viewBox="0 0 24 24"
            style={{ marginRight: "0.25rem" }}
          >
            <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l.73-2.19"></path>
          </svg>
          Refresh Logs
        </button>
      </div>

      {logs.length === 0 ? (
        <div className="glass-card" style={{ padding: "3rem", textAlign: "center" }}>
          <svg width="40" height="40" fill="none" stroke="var(--text-muted)" strokeWidth="1.5" viewBox="0 0 24 24" style={{ marginBottom: "1rem" }}>
            <path d="M12 8v4l3 3M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
          <p style={{ color: "var(--text-muted)", fontSize: "0.95rem" }}>No update actions have been recorded yet.</p>
        </div>
      ) : (
        <div className="table-wrapper">
          <table className="custom-table">
            <thead>
              <tr>
                <th>Job ID</th>
                <th>Workload Name</th>
                <th>Status</th>
                <th>Duration</th>
                <th>Executed At</th>
                <th>Logs / Details</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td style={{ fontFamily: "monospace", fontSize: "0.85rem", color: "var(--text-secondary)" }}>
                    {log.id}
                  </td>
                  <td style={{ fontWeight: "500" }}>{log.workload_name}</td>
                  <td>
                    <span className={`status-badge ${getStatusClass(log.status)}`}>
                      {log.status.replace("_", " ")}
                    </span>
                  </td>
                  <td style={{ color: "var(--text-secondary)" }}>
                    {log.status === "in_progress" || log.status === "pending"
                      ? "Running..."
                      : getDuration(log.created_at, log.updated_at)}
                  </td>
                  <td style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
                    {new Date(log.created_at).toLocaleString()}
                  </td>
                  <td style={{ fontSize: "0.88rem" }}>
                    {log.status === "failed" && log.error_message ? (
                      <span style={{ color: "var(--color-error)", fontStyle: "italic" }} title={log.error_message}>
                        {log.error_message.length > 50
                          ? `${log.error_message.substring(0, 50)}...`
                          : log.error_message}
                      </span>
                    ) : log.status === "rolled_back" && log.error_message ? (
                      <span style={{ color: "var(--color-warning)", fontStyle: "italic" }} title={log.error_message}>
                        {log.error_message}
                      </span>
                    ) : (
                      <span style={{ color: "var(--text-muted)" }}>Recreation successful</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
