import React, { useState } from "react";
import { Workload } from "../services/api";
import WorkloadItem from "./WorkloadItem";

interface DashboardProps {
  workloads: Workload[];
  onTriggerUpdate: (id: string) => void;
  isChecking: boolean;
  onManualCheck: () => void;
  jobProgress?: Record<string, { percent?: number; message?: string; podStatus?: string }>;
}

export default function Dashboard({
  workloads,
  onTriggerUpdate,
  isChecking,
  onManualCheck,
  jobProgress = {},
}: DashboardProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const [orchestratorFilter, setOrchestratorFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  // Extract unique projects/namespaces for filtering
  const projects = Array.from(new Set(workloads.map((w) => w.namespace_project)));

  // Filter logic
  const filteredWorkloads = workloads.filter((w) => {
    const matchesSearch =
      w.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      w.current_image.toLowerCase().includes(searchTerm.toLowerCase()) ||
      w.namespace_project.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesOrchestrator =
      orchestratorFilter === "all" || w.orchestrator_type === orchestratorFilter;

    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "update_available" && w.update_status === "update_available") ||
      (statusFilter === "up_to_date" && w.update_status === "up_to_date") ||
      (statusFilter === "updating" && w.update_status === "updating") ||
      (statusFilter === "failed" && w.update_status === "failed");

    return matchesSearch && matchesOrchestrator && matchesStatus;
  });

  const updatesCount = workloads.filter((w) => w.update_status === "update_available").length;

  return (
    <div className="dashboard-section">
      {/* Search and Filters Bar */}
      <div className="filter-bar glass-card" style={{ marginBottom: "2rem", display: "flex", flexWrap: "wrap", gap: "1rem", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", gap: "1rem", flexGrow: 1, flexBasis: "400px" }}>
          <div style={{ position: "relative", width: "100%" }}>
            <input
              type="text"
              placeholder="Search workloads, images, namespace..."
              className="form-control"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              style={{ paddingLeft: "2.5rem" }}
            />
            <span style={{ position: "absolute", left: "0.85rem", top: "50%", transform: "translateY(-50%)", color: "var(--text-muted)" }}>
              <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
            </span>
          </div>
        </div>

        <div style={{ display: "flex", gap: "0.75rem", flexWrap: "wrap" }}>
          {/* Orchestrator Filter */}
          <select
            className="form-control"
            value={orchestratorFilter}
            onChange={(e) => setOrchestratorFilter(e.target.value)}
            style={{ width: "150px" }}
          >
            <option value="all">All Runtimes</option>
            <option value="compose">Compose</option>
            <option value="kubernetes">Kubernetes</option>
          </select>

          {/* Status Filter */}
          <select
            className="form-control"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            style={{ width: "160px" }}
          >
            <option value="all">All Statuses</option>
            <option value="update_available">Update Available ({updatesCount})</option>
            <option value="up_to_date">Up to Date</option>
            <option value="updating">Updating</option>
            <option value="failed">Failed</option>
          </select>

          {/* Trigger Scan Button */}
          <button
            className="btn btn-secondary"
            onClick={onManualCheck}
            disabled={isChecking}
            title="Scan registries immediately"
          >
            <svg
              className={isChecking ? "spin" : ""}
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
            {isChecking ? "Scanning..." : "Scan Now"}
          </button>
        </div>
      </div>

      {/* Grid of Workloads */}
      {filteredWorkloads.length === 0 ? (
        <div className="glass-card" style={{ padding: "4rem", textAlign: "center", borderStyle: "dashed" }}>
          <svg width="48" height="48" fill="none" stroke="var(--text-muted)" strokeWidth="1.5" viewBox="0 0 24 24" style={{ marginBottom: "1rem" }}>
            <rect x="2" y="2" width="20" height="20" rx="2" ry="2"></rect>
            <path d="M12 8v4M12 16h.01"></path>
          </svg>
          <h3 style={{ color: "var(--text-secondary)", marginBottom: "0.5rem" }}>No Monitored Workloads Found</h3>
          <p style={{ color: "var(--text-muted)", fontSize: "0.95rem" }}>
            Make sure to add the label `container-updater.enable=true` to your containers (Docker Compose) or annotation `container-updater.enable: "true"` (Kubernetes).
          </p>
        </div>
      ) : (
        <div className="workload-grid" style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(360px, 1fr))", gap: "1.5rem" }}>
          {filteredWorkloads.map((w) => (
            <WorkloadItem
              key={w.id}
              workload={w}
              onTriggerUpdate={onTriggerUpdate}
              progress={jobProgress[w.id]}
            />
          ))}
        </div>
      )}
    </div>
  );
}
