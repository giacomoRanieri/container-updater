import React, { useState, useEffect, useCallback } from "react";
import Head from "next/head";
import { fetchSession, redirectToLogin, User } from "../services/auth";
import {
  fetchWorkloads,
  fetchAuditLogs,
  fetchStats,
  triggerWorkloadUpdate,
  triggerManualScan,
  connectWebSocket,
  Workload,
  AuditLogItem,
  AppStats,
} from "../services/api";
import Dashboard from "../components/Dashboard";
import AuditLog from "../components/AuditLog";

export default function Home() {
  const [sessionLoading, setSessionLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);

  // Application Data States
  const [workloads, setWorkloads] = useState<Workload[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLogItem[]>([]);
  const [stats, setStats] = useState<AppStats>({
    total_workloads: 0,
    updates_available: 0,
    successful_updates: 0,
  });

  const [activeTab, setActiveTab] = useState<"workloads" | "logs">("workloads");
  const [isDataLoading, setIsDataLoading] = useState(false);
  const [isScanning, setIsScanning] = useState(false);

  // Authenticate user session
  useEffect(() => {
    fetchSession().then((session) => {
      if (session.authenticated && session.user) {
        setIsAuthenticated(true);
        setCurrentUser(session.user);
      } else {
        setIsAuthenticated(false);
        setCurrentUser(null);
      }
      setSessionLoading(false);
    });
  }, []);

  // Fetch all dashboard data
  const refreshAllData = useCallback(async () => {
    if (!isAuthenticated) return;
    setIsDataLoading(true);
    try {
      const [wList, aLogs, appStats] = await Promise.all([
        fetchWorkloads(),
        fetchAuditLogs(),
        fetchStats(),
      ]);
      setWorkloads(wList);
      setAuditLogs(aLogs);
      setStats(appStats);
    } catch (err) {
      console.error("Failed to load dashboard data:", err);
    } finally {
      setIsDataLoading(false);
    }
  }, [isAuthenticated]);

  // Load data once authenticated
  useEffect(() => {
    if (isAuthenticated) {
      refreshAllData();
    }
  }, [isAuthenticated, refreshAllData]);

  const [jobProgress, setJobProgress] = useState<Record<string, { percent?: number; message?: string; podStatus?: string }>>({});

  // Connect WebSocket for real-time updates
  useEffect(() => {
    if (!isAuthenticated) return;

    const cleanup = connectWebSocket((event, data) => {
      if (event === "workload_updated") {
        // Update specific workload status dynamically in the list
        setWorkloads((prev) =>
          prev.map((w) => (w.id === data.id ? { ...w, update_status: data.update_status } : w))
        );
        // Refresh stats
        fetchStats().then(setStats).catch(console.error);
      } else if (event === "job_status") {
        if (data.workload_id) {
          setJobProgress((prev) => ({
            ...prev,
            [data.workload_id]: {
              percent: data.percent_complete,
              message: data.message,
            },
          }));
        }
        // If a job completes or fails, refresh audit logs and workloads
        if (data.status === "completed" || data.status === "failed" || data.status === "rolled_back") {
          if (data.workload_id) {
            setJobProgress((prev) => {
              const copy = { ...prev };
              delete copy[data.workload_id];
              return copy;
            });
          }
          refreshAllData();
        }
      } else if (event === "pod_event") {
        if (data.workload_id) {
          setJobProgress((prev) => ({
            ...prev,
            [data.workload_id]: {
              ...prev[data.workload_id],
              podStatus: data.status,
              message: data.message || prev[data.workload_id]?.message,
            },
          }));
        }
      }
    });

    return cleanup;
  }, [isAuthenticated, refreshAllData]);

  // Trigger manual update
  const handleTriggerUpdate = async (id: string) => {
    try {
      // Optimitic UI state update
      setWorkloads((prev) =>
        prev.map((w) => (w.id === id ? { ...w, update_status: "updating" } : w))
      );

      const result = await triggerWorkloadUpdate(id);
      console.log("Update triggered successfully:", result);
    } catch (err) {
      console.error("Failed to trigger update:", err);
      // Revert workloads list state on failure
      refreshAllData();
    }
  };

  // Trigger manual registry scan
  const handleManualCheck = async () => {
    setIsScanning(true);
    try {
      await triggerManualScan();
      setTimeout(() => {
        refreshAllData();
      }, 1500);
    } catch (err) {
      console.error("Failed to run manual check:", err);
    } finally {
      setIsScanning(false);
    }
  };

  // Login page layout
  if (sessionLoading) {
    return (
      <div style={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "100vh", backgroundColor: "var(--bg-primary)" }}>
        <svg className="spin" width="48" height="48" fill="none" stroke="var(--accent-indigo)" strokeWidth="3" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" strokeDasharray="30 10"></circle>
        </svg>
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="app-container" style={{ justifyContent: "center", alignItems: "center" }}>
        <Head>
          <title>Login | Container Updater</title>
          <meta name="description" content="Securely access Container Updater console" />
        </Head>

        <div className="glass-card" style={{ maxWidth: "460px", width: "90%", padding: "3rem 2.5rem", textAlign: "center", display: "flex", flexDirection: "column", gap: "1.5rem" }}>
          
          {/* Logo and title */}
          <div>
            <div style={{ background: "var(--accent-gradient)", width: "56px", height: "56px", borderRadius: "14px", display: "inline-flex", alignItems: "center", justifyContent: "center", color: "white", marginBottom: "1rem", boxShadow: "0 8px 24px rgba(99, 102, 241, 0.3)" }}>
              <svg width="28" height="28" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
              </svg>
            </div>
            <h1 style={{ fontSize: "2rem", marginBottom: "0.25rem" }}>
              Container <span className="text-gradient">Updater</span>
            </h1>
            <p style={{ color: "var(--text-muted)", fontSize: "0.95rem" }}>
              Automated image checking, registry monitoring, and rolling workload restarts.
            </p>
          </div>

          <div style={{ borderTop: "1px solid var(--glass-border)", paddingTop: "1.5rem" }}>
            <button className="btn btn-primary" onClick={redirectToLogin} style={{ width: "100%", padding: "0.85rem", fontSize: "1rem" }}>
              <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24" style={{ marginRight: "0.5rem" }}>
                <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3"></path>
              </svg>
              Sign In with OIDC
            </button>
          </div>

          <div style={{ fontSize: "0.8rem", color: "var(--text-muted)" }}>
            Supports secure multi-user OIDC providers including Authentik, Authelia, Keycloak.
          </div>
        </div>
      </div>
    );
  }

  // Dashboard landing page layout
  return (
    <div className="app-container">
      <Head>
        <title>Dashboard | Container Updater</title>
        <meta name="description" content="Monitor and update containers easily" />
      </Head>

      {/* Navigation Header */}
      <header className="navbar">
        <div className="navbar-content">
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
            <div style={{ background: "var(--accent-gradient)", width: "32px", height: "32px", borderRadius: "8px", display: "inline-flex", alignItems: "center", justifyContent: "center", color: "white" }}>
              <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
              </svg>
            </div>
            <span style={{ fontSize: "1.2rem", fontWeight: "600" }}>
              Container <span className="text-gradient">Updater</span>
            </span>
          </div>

          <div className="nav-links">
            <span
              className={`nav-link ${activeTab === "workloads" ? "active" : ""}`}
              onClick={() => setActiveTab("workloads")}
            >
              Workloads
            </span>
            <span
              className={`nav-link ${activeTab === "logs" ? "active" : ""}`}
              onClick={() => setActiveTab("logs")}
            >
              Audit Log
            </span>
            <a href="/settings" className="nav-link">
              Settings
            </a>
            
            {currentUser && (
              <div className="user-badge">
                <svg width="14" height="14" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z"/>
                </svg>
                <span>{currentUser.username}</span>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main console layout */}
      <main className="main-content">
        
        {/* App Stats Overview Row */}
        <section style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))", gap: "1.5rem", marginBottom: "2rem" }}>
          
          <div className="glass-card" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <div>
              <p style={{ color: "var(--text-muted)", fontSize: "0.9rem", marginBottom: "0.25rem" }}>Monitored Containers</p>
              <h3 style={{ fontSize: "2rem" }}>{stats.total_workloads}</h3>
            </div>
            <div style={{ padding: "0.75rem", borderRadius: "10px", background: "rgba(99, 102, 241, 0.1)", color: "var(--accent-indigo)" }}>
              <svg width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <rect x="2" y="2" width="20" height="20" rx="2" ry="2"></rect>
                <path d="M9 2v20M15 2v20M2 9h20M2 15h20"></path>
              </svg>
            </div>
          </div>

          <div className="glass-card" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <div>
              <p style={{ color: "var(--text-muted)", fontSize: "0.9rem", marginBottom: "0.25rem" }}>Updates Available</p>
              <h3 style={{ fontSize: "2rem", color: stats.updates_available > 0 ? "var(--color-warning)" : "inherit" }}>
                {stats.updates_available}
              </h3>
            </div>
            <div style={{ padding: "0.75rem", borderRadius: "10px", background: "rgba(245, 158, 11, 0.1)", color: "var(--color-warning)" }}>
              <svg width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"></path>
                <path d="M12 8v4l3 3"></path>
              </svg>
            </div>
          </div>

          <div className="glass-card" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <div>
              <p style={{ color: "var(--text-muted)", fontSize: "0.9rem", marginBottom: "0.25rem" }}>Successful Updates</p>
              <h3 style={{ fontSize: "2rem" }}>{stats.successful_updates}</h3>
            </div>
            <div style={{ padding: "0.75rem", borderRadius: "10px", background: "rgba(16, 185, 129, 0.1)", color: "var(--color-success)" }}>
              <svg width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                <polyline points="22 4 12 14.01 9 11.01"></polyline>
              </svg>
            </div>
          </div>

        </section>

        {/* Tab content area */}
        {activeTab === "workloads" ? (
          <Dashboard
            workloads={workloads}
            onTriggerUpdate={handleTriggerUpdate}
            isChecking={isScanning}
            onManualCheck={handleManualCheck}
            jobProgress={jobProgress}
          />
        ) : (
          <AuditLog
            logs={auditLogs}
            onRefresh={refreshAllData}
            isLoading={isDataLoading}
          />
        )}

      </main>

      {/* Console Footer */}
      <footer style={{ borderTop: "1px solid var(--glass-border)", padding: "1.5rem", textAlign: "center", color: "var(--text-muted)", fontSize: "0.85rem", background: "var(--bg-secondary)" }}>
        Container Updater Console &bull; Local Server Time: {new Date().toLocaleDateString()}
      </footer>
    </div>
  );
}
