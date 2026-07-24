import React from "react";
import { Workload } from "../services/api";

interface WorkloadItemProps {
  workload: Workload;
  onTriggerUpdate: (id: string) => void;
  progress?: {
    percent?: number;
    message?: string;
    podStatus?: string;
  };
}

export default function WorkloadItem({ workload, onTriggerUpdate, progress }: WorkloadItemProps) {
  const isUpdating = workload.update_status === "updating";

  return (
    <div
      className="glass-card workload-card"
      style={{
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        position: "relative",
        overflow: "hidden",
      }}
    >
      <div>
        {/* Header Name & Orchestrator Type Icon & Status Badge */}
        <div
          style={{
            display: "flex",
            alignItems: "flex-start",
            justifyContent: "space-between",
            gap: "0.75rem",
            marginBottom: "1rem",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", minWidth: 0, flex: 1 }}>
            {workload.orchestrator_type === "compose" ? (
              <div
                style={{
                  background: "rgba(36, 150, 237, 0.15)",
                  color: "#2496ed",
                  padding: "0.45rem",
                  borderRadius: "8px",
                  display: "flex",
                  flexShrink: 0,
                }}
                title="Docker Compose"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M13.983 11.078h2.119c.102 0 .186-.084.186-.186V8.773c0-.102-.084-.186-.186-.186h-2.119c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m-2.95 0h2.118c.103 0 .187-.084.187-.186V8.773c0-.102-.084-.186-.187-.186h-2.118c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m-2.95 0h2.119c.102 0 .185-.084.185-.186V8.773c0-.102-.083-.186-.185-.186H8.083c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m-2.949 0h2.119c.102 0 .185-.084.185-.186V8.773c0-.102-.083-.186-.185-.186H5.134c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m2.949-2.95h2.119c.102 0 .185-.084.185-.186V5.823c0-.102-.083-.186-.185-.186H8.083c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m-2.949 0h2.119c.102 0 .185-.084.185-.186V5.823c0-.102-.083-.186-.185-.186H5.134c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m5.899 0h2.118c.103 0 .187-.084.187-.186V5.823c0-.102-.084-.186-.187-.186h-2.118c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m-8.848 5.9h2.119c.102 0 .186-.084.186-.186v-2.12c0-.102-.084-.185-.186-.185H2.185c-.102 0-.186.083-.186.185v2.12c0 .102.084.186.186.186m2.949 2.949h2.119c.102 0 .185-.084.185-.186v-2.119c0-.102-.083-.186-.185-.186H5.134c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m5.899 0h2.118c.103 0 .187-.084.187-.186v-2.119c0-.102-.084-.186-.187-.186h-2.118c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m-2.95 0h2.119c.102 0 .185-.084.185-.186v-2.119c0-.102-.083-.186-.185-.186H8.083c-.102 0-.186.084-.186.186v2.119c0 .102.084.186.186.186m11.899-5.9h-4.06c-.452 0-.822-.37-.822-.823V4.934c0-.071.04-.137.104-.17a.187.187 0 0 1 .199.01l4.908 3.329c.196.133.247.4.114.596a.434.434 0 0 1-.343.178M23.632 12c.007-.066.007-.133.007-.2 0-3.64-2.825-6.623-6.425-6.883-.102-.007-.186.069-.193.171l-.105 1.542c-.007.1.069.186.171.193 2.766.19 4.936 2.457 4.936 5.213a5.2 5.2 0 0 1-.502 2.228.188.188 0 0 0 .093.245l1.411.666a.187.187 0 0 0 .252-.082c.237-.432.423-.896.552-1.378l.004-.008zM1.986 16.486c.076.066.189.043.238-.046a7.63 7.63 0 0 0 .809-2.227c.026-.145-.078-.282-.224-.294A9.85 9.85 0 0 1 .006 12a9.92 9.92 0 0 1 1.98-6.084c.059-.08.037-.193-.046-.247l-1.282-.843a.188.188 0 0 0-.256.046C.139 5.34-.006 6.014-.006 6.702c0 .248.02.493.056.732a.187.187 0 0 0 .21.157l1.545-.126c.101-.008.186.069.194.17 0 .02.007.039.007.059 0 3.738 2.871 6.8 6.558 7.086.102.008.186-.068.194-.17l.156-2.12c.008-.102-.068-.186-.17-.194c-2.482-.18-4.444-2.197-4.522-4.686-.003-.102-.087-.183-.19-.183H2.06c-.102 0-.185.082-.186.185a7.35 7.35 0 0 0 .428 2.502.188.188 0 0 0 .235.122c.3-.087.61-.137.925-.152.102-.005.188.072.193.174l.115 2.115c.006.102-.072.188-.174.193a9.8 9.8 0 0 1-1.398.026c-.1.011-.181.096-.173.197.042.544.137 1.077.283 1.589z"/>
                </svg>
              </div>
            ) : (
              <div
                style={{
                  background: "rgba(50, 108, 230, 0.15)",
                  color: "#326ce5",
                  padding: "0.45rem",
                  borderRadius: "8px",
                  display: "flex",
                  flexShrink: 0,
                }}
                title="Kubernetes"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 0c-.172 0-.34.048-.485.138L1.644 6.223a.965.965 0 0 0-.486.837v11.88c0 .339.176.654.469.836l9.907 6.084a.97.97 0 0 0 .964 0l9.871-6.084c.29-.181.467-.496.467-.835V7.06a.964.964 0 0 0-.485-.836l-9.923-6.085c-.145-.09-.313-.138-.485-.138zm.482 2.664l7.653 4.695-2.614 4.542h-.002l-5.037-2.923v-6.314zm-.964 0v6.314l-5.038 2.923L3.86 7.359l7.657-4.695zm-.482 7.74l4.57 2.651v5.276l-4.57-2.651v-5.276zm1.928 0l4.57 2.651v5.276l-4.57-2.651v-5.276zM3.107 8.665l2.615 4.541-2.615 4.541v-9.082zm17.786 0v9.082l-2.615-4.541 2.615-4.541zM7.054 15.5l5.038-2.923v5.845l-5.038-2.922zm9.892 0l-5.038 2.922V12.58l5.038 2.923zM3.861 16.641l7.657 4.695v-6.314l-5.038-2.923-2.619 4.542zm16.278 0l-2.62-4.542-5.037 2.923v6.314l7.657-4.695z"/>
                </svg>
              </div>
            )}
            <div style={{ minWidth: 0, flex: 1 }}>
              <h3
                style={{
                  fontSize: "1.1rem",
                  fontWeight: "600",
                  margin: 0,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
                title={workload.name}
              >
                {workload.name}
              </h3>
              <p style={{ fontSize: "0.8rem", color: "var(--text-muted)", margin: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                Project: <span style={{ color: "var(--text-secondary)" }}>{workload.namespace_project}</span>
              </p>
            </div>
          </div>

          <div style={{ flexShrink: 0 }}>
            <span className={`status-badge status-${workload.update_status}`}>
              {workload.update_status.replace("_", " ")}
            </span>
          </div>
        </div>

        {/* Details Section */}
        <div
          className="workload-details"
          style={{
            display: "flex",
            flexDirection: "column",
            gap: "0.6rem",
            fontSize: "0.88rem",
            background: "rgba(255,255,255,0.02)",
            padding: "0.85rem",
            borderRadius: "8px",
            border: "1px solid var(--glass-border)",
            marginBottom: "1rem",
          }}
        >
          <div style={{ display: "flex", justifyContent: "space-between" }}>
            <span style={{ color: "var(--text-secondary)" }}>Current Tag:</span>
            <span style={{ fontFamily: "monospace", color: "var(--text-primary)", fontWeight: "500" }}>
              {workload.current_image.split(":")[1] || "latest"}
            </span>
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: "0.15rem" }}>
            <span style={{ color: "var(--text-secondary)" }}>Running Image:</span>
            <span
              style={{
                fontFamily: "monospace",
                color: "var(--text-muted)",
                fontSize: "0.8rem",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
              title={workload.current_image}
            >
              {workload.current_image}
            </span>
          </div>
          <div style={{ display: "flex", justifyContent: "space-between" }}>
            <span style={{ color: "var(--text-secondary)" }}>Digest:</span>
            <span style={{ fontFamily: "monospace", color: "var(--text-muted)", fontSize: "0.8rem" }}>
              {workload.current_digest.substring(0, 15)}...
            </span>
          </div>

          {workload.update_status === "update_available" && workload.new_digest && (
            <div style={{ marginTop: "0.5rem", borderTop: "1px dashed var(--glass-border)", paddingTop: "0.5rem" }}>
              {(workload.new_image?.split(":")[1] || "latest") !== (workload.current_image.split(":")[1] || "latest") && (
                <div style={{ display: "flex", justifyContent: "space-between", color: "var(--color-warning)" }}>
                  <span>New Available Tag:</span>
                  <span style={{ fontFamily: "monospace" }}>
                    {workload.new_image?.split(":")[1] || "latest"}
                  </span>
                </div>
              )}
              <div style={{ display: "flex", justifyContent: "space-between", color: "var(--color-warning)", marginTop: "0.2rem" }}>
                <span>New Available Digest:</span>
                <span style={{ fontFamily: "monospace" }}>
                  {workload.new_digest.substring(0, 15)}...
                </span>
              </div>
            </div>
          )}

          {isUpdating && (
            <div style={{ marginTop: "0.5rem", borderTop: "1px dashed var(--glass-border)", paddingTop: "0.5rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between", fontSize: "0.8rem", color: "#38bdf8", marginBottom: "0.35rem" }}>
                <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", marginRight: "0.5rem" }}>
                  {progress?.message || "Rolling out update..."}
                </span>
                {progress?.podStatus && (
                  <span style={{ fontWeight: 600, fontFamily: "monospace", color: "#a78bfa" }}>
                    [{progress.podStatus}]
                  </span>
                )}
              </div>
              <div style={{ width: "100%", background: "rgba(255, 255, 255, 0.1)", borderRadius: "4px", height: "6px", overflow: "hidden" }}>
                <div
                  style={{
                    width: `${progress?.percent || 40}%`,
                    background: "linear-gradient(90deg, #38bdf8, #818cf8)",
                    height: "100%",
                    borderRadius: "4px",
                    transition: "width 0.4s ease",
                  }}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Card Footer Actions & Timestamps */}
      <div>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontSize: "0.75rem",
            color: "var(--text-muted)",
            marginBottom: "1rem",
          }}
        >
          <span>Checked: {workload.last_checked_at ? new Date(workload.last_checked_at).toLocaleTimeString() : "Never"}</span>
          {workload.last_updated_at && (
            <span>Updated: {new Date(workload.last_updated_at).toLocaleDateString()}</span>
          )}
        </div>

        <div style={{ display: "flex", gap: "0.5rem" }}>
          {workload.update_status === "update_available" || workload.update_status === "failed" ? (
            <button
              className="btn btn-primary"
              onClick={() => onTriggerUpdate(workload.id)}
              style={{ width: "100%", padding: "0.5rem" }}
            >
              <svg
                width="16"
                height="16"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                viewBox="0 0 24 24"
                style={{ marginRight: "0.25rem" }}
              >
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"></path>
              </svg>
              Execute Update
            </button>
          ) : isUpdating ? (
            <button className="btn btn-secondary" disabled style={{ width: "100%", padding: "0.5rem", color: "#38bdf8" }}>
              <svg
                className="spin"
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
              {progress?.percent ? `Updating (${progress.percent}%)` : "Updating..."}
            </button>
          ) : (
            <button
              className="btn btn-secondary"
              disabled
              style={{ width: "100%", padding: "0.5rem", opacity: "0.7" }}
            >
              <svg
                width="16"
                height="16"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                viewBox="0 0 24 24"
                style={{ marginRight: "0.25rem" }}
              >
                <path d="M20 6L9 17l-5-5"></path>
              </svg>
              Up to Date
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
