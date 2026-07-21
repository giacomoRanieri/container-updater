import React, { useState, useEffect } from "react";
import Head from "next/head";
import Link from "next/link";
import {
  fetchNotifications,
  saveNotification,
  NotificationService,
  fetchRegistries,
  saveRegistry,
  deleteRegistry,
  RegistryCredential,
} from "../services/api";

export default function Settings() {
  const [notifications, setNotifications] = useState<NotificationService[]>([]);
  const [registries, setRegistries] = useState<RegistryCredential[]>([]);
  const [activeTab, setActiveTab] = useState<"notifications" | "registries" | "system">("notifications");
  const [isLoading, setIsLoading] = useState(false);
  const [isRegistrySaving, setIsRegistrySaving] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // New notification service form state
  const [formName, setFormName] = useState("");
  const [formUrl, setFormUrl] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);
  const [editingId, setEditingId] = useState<string | null>(null);

  const [message, setMessage] = useState<{ text: string; type: "success" | "error" } | null>(null);

  const [registryServer, setRegistryServer] = useState("");
  const [registryUsername, setRegistryUsername] = useState("");
  const [registryPassword, setRegistryPassword] = useState("");

  // Load notification settings
  const loadSettings = async () => {
    setIsLoading(true);
    try {
      const list = await fetchNotifications();
      setNotifications(list);
    } catch (err) {
      console.error("Failed to load notification settings", err);
      showMsg("Failed to retrieve notification configurations", "error");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadSettings();
  }, []);

  const loadRegistries = async () => {
    try {
      const list = await fetchRegistries();
      setRegistries(list);
    } catch (err) {
      console.error("Failed to load registry credentials", err);
      showMsg("Failed to retrieve registry credentials", "error");
    }
  };

  useEffect(() => {
    if (activeTab === "registries") {
      loadRegistries();
    }
  }, [activeTab]);

  const showMsg = (text: string, type: "success" | "error") => {
    setMessage({ text, type });
    setTimeout(() => setMessage(null), 5000);
  };

  // Handle Form Submission
  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formName || !formUrl) {
      showMsg("Name and Apprise URL are required fields", "error");
      return;
    }

    setIsSaving(true);
    try {
      const payload: Omit<NotificationService, "id"> & { id?: string } = {
        name: formName,
        apprise_url: formUrl,
        is_enabled: formEnabled,
      };

      if (editingId) {
        payload.id = editingId;
      }

      await saveNotification(payload);
      showMsg("Notification settings saved successfully", "success");
      
      // Reset form
      setFormName("");
      setFormUrl("");
      setFormEnabled(true);
      setEditingId(null);
      
      // Reload list
      loadSettings();
    } catch (err) {
      console.error("Failed to save settings", err);
      showMsg("Failed to save notification service", "error");
    } finally {
      setIsSaving(false);
    }
  };

  const handleEdit = (service: NotificationService) => {
    setEditingId(service.id);
    setFormName(service.name);
    setFormUrl(service.apprise_url);
    setFormEnabled(service.is_enabled);
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setFormName("");
    setFormUrl("");
    setFormEnabled(true);
  };

  const handleRegistrySave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!registryServer || !registryUsername || !registryPassword) {
      showMsg("Server address, username, and password are required", "error");
      return;
    }

    setIsRegistrySaving(true);
    try {
      await saveRegistry({
        server_address: registryServer,
        username: registryUsername,
        password: registryPassword,
      });
      showMsg("Registry credentials saved successfully", "success");
      setRegistryServer("");
      setRegistryUsername("");
      setRegistryPassword("");
      loadRegistries();
    } catch (err) {
      console.error("Failed to save registry credential", err);
      showMsg("Failed to save registry credential", "error");
    } finally {
      setIsRegistrySaving(false);
    }
  };

  const handleDeleteRegistry = async (id: string) => {
    try {
      await deleteRegistry(id);
      showMsg("Registry credential deleted", "success");
      loadRegistries();
    } catch (err) {
      console.error("Failed to delete registry credential", err);
      showMsg("Failed to delete registry credential", "error");
    }
  };

  return (
    <div className="app-container">
      <Head>
        <title>Settings | Container Updater</title>
        <meta name="description" content="Configure notifications and integrations" />
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
            <Link href="/" className="nav-link">
              Dashboard
            </Link>
            <Link href="/" className="nav-link">
              Audit Log
            </Link>
            <span className="nav-link active">Settings</span>
          </div>
        </div>
      </header>

      {/* Settings Main Layout */}
      <main className="main-content" style={{ display: "grid", gridTemplateColumns: "240px 1fr", gap: "2rem", alignItems: "start" }}>
        
        {/* Settings Sidebar Tabs */}
        <aside className="glass-card" style={{ padding: "1rem", display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          <button
            className="btn nav-link"
            style={{
              justifyContent: "flex-start",
              width: "100%",
              background: activeTab === "notifications" ? "rgba(255,255,255,0.05)" : "transparent",
              color: activeTab === "notifications" ? "var(--text-primary)" : "var(--text-secondary)",
              padding: "0.75rem 1rem",
              borderRadius: "8px",
            }}
            onClick={() => setActiveTab("notifications")}
          >
            <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24" style={{ marginRight: "0.5rem" }}>
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0"></path>
            </svg>
            Apprise Notifications
          </button>
          <button
            className="btn nav-link"
            style={{
              justifyContent: "flex-start",
              width: "100%",
              background: activeTab === "registries" ? "rgba(255,255,255,0.05)" : "transparent",
              color: activeTab === "registries" ? "var(--text-primary)" : "var(--text-secondary)",
              padding: "0.75rem 1rem",
              borderRadius: "8px",
            }}
            onClick={() => setActiveTab("registries")}
          >
            <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24" style={{ marginRight: "0.5rem" }}>
              <rect x="3" y="4" width="18" height="16" rx="3"></rect>
              <path d="M7 8h10M7 12h10M7 16h6"></path>
            </svg>
            Registries
          </button>
          <button
            className="btn nav-link"
            style={{
              justifyContent: "flex-start",
              width: "100%",
              background: activeTab === "system" ? "rgba(255,255,255,0.05)" : "transparent",
              color: activeTab === "system" ? "var(--text-primary)" : "var(--text-secondary)",
              padding: "0.75rem 1rem",
              borderRadius: "8px",
            }}
            onClick={() => setActiveTab("system")}
          >
            <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24" style={{ marginRight: "0.5rem" }}>
              <rect x="2" y="2" width="20" height="20" rx="5" ry="5"></rect>
              <path d="M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37zM17.5 6.5h.01"></path>
            </svg>
            GitOps & System Status
          </button>
        </aside>

        {/* Settings Tab Content */}
        <section style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
          
          {message && (
            <div
              className="glass-card"
              style={{
                borderColor: message.type === "success" ? "var(--color-success)" : "var(--color-error)",
                background: message.type === "success" ? "rgba(16,185,129,0.08)" : "rgba(239,68,68,0.08)",
                color: message.type === "success" ? "var(--color-success)" : "var(--color-error)",
                padding: "0.85rem 1.25rem",
                borderRadius: "8px",
                fontSize: "0.95rem",
              }}
            >
              {message.text}
            </div>
          )}

          {activeTab === "notifications" ? (
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1.5rem", alignItems: "start" }}>
              {/* Form card */}
              <div className="glass-card">
                <h2 style={{ fontSize: "1.2rem", marginBottom: "1.25rem" }}>
                  {editingId ? "Modify Notification Endpoint" : "Add Notification Endpoint"}
                </h2>
                
                <form onSubmit={handleSave} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label>Configuration Name</label>
                    <input
                      type="text"
                      className="form-control"
                      placeholder="e.g. Admin Telegram Chat"
                      value={formName}
                      onChange={(e) => setFormName(e.target.value)}
                    />
                  </div>

                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label>Apprise Connection URL</label>
                    <input
                      type="text"
                      className="form-control"
                      placeholder="tgram://botToken/chatID"
                      value={formUrl}
                      onChange={(e) => setFormUrl(e.target.value)}
                    />
                    <small style={{ color: "var(--text-muted)", fontSize: "0.78rem", marginTop: "0.3rem", display: "block" }}>
                      Apprise format examples: `tgram://bottoken/chatid` or `pushed://appkey`
                    </small>
                  </div>

                  <div className="form-checkbox">
                    <input
                      type="checkbox"
                      id="is-enabled-chk"
                      checked={formEnabled}
                      onChange={(e) => setFormEnabled(e.target.checked)}
                    />
                    <label htmlFor="is-enabled-chk">Enable this notification channel</label>
                  </div>

                  <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.5rem" }}>
                    <button
                      type="submit"
                      className="btn btn-primary"
                      disabled={isSaving}
                      style={{ flexGrow: 1 }}
                    >
                      {isSaving ? "Saving..." : "Save Configuration"}
                    </button>
                    {editingId && (
                      <button
                        type="button"
                        className="btn btn-secondary"
                        onClick={handleCancelEdit}
                      >
                        Cancel
                      </button>
                    )}
                  </div>
                </form>
              </div>

              {/* Endpoints List card */}
              <div className="glass-card" style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
                <h2 style={{ fontSize: "1.2rem" }}>Configured Channels</h2>

                {isLoading ? (
                  <div style={{ padding: "2rem", textAlign: "center" }}>
                    <svg className="spin" width="24" height="24" fill="none" stroke="var(--accent-indigo)" strokeWidth="2" viewBox="0 0 24 24">
                      <circle cx="12" cy="12" r="10" strokeDasharray="15 5"></circle>
                    </svg>
                  </div>
                ) : notifications.length === 0 ? (
                  <div style={{ padding: "2rem", textAlign: "center", color: "var(--text-muted)", fontSize: "0.9rem" }}>
                    No notification channels registered.
                  </div>
                ) : (
                  <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
                    {notifications.map((n) => (
                      <div
                        key={n.id}
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          alignItems: "center",
                          padding: "0.85rem",
                          background: "rgba(255,255,255,0.02)",
                          borderRadius: "8px",
                          border: "1px solid var(--glass-border)",
                        }}
                      >
                        <div style={{ overflow: "hidden", paddingRight: "1rem" }}>
                          <h4 style={{ fontSize: "0.95rem", margin: 0, display: "flex", alignItems: "center", gap: "0.5rem" }}>
                            {n.name}
                            <span
                              style={{
                                width: "6px",
                                height: "6px",
                                borderRadius: "50%",
                                background: n.is_enabled ? "var(--color-success)" : "var(--color-error)",
                              }}
                            />
                          </h4>
                          <p
                            style={{
                              fontFamily: "monospace",
                              fontSize: "0.75rem",
                              color: "var(--text-muted)",
                              margin: "0.2rem 0 0 0",
                              overflow: "hidden",
                              textOverflow: "ellipsis",
                              whiteSpace: "nowrap",
                            }}
                          >
                            {n.apprise_url}
                          </p>
                        </div>
                        <button
                          className="btn btn-secondary"
                          style={{ padding: "0.35rem 0.65rem", fontSize: "0.82rem" }}
                          onClick={() => handleEdit(n)}
                        >
                          Modify
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : activeTab === "registries" ? (
            <div className="glass-card" style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
              <div>
                <h2 style={{ fontSize: "1.2rem", marginBottom: "0.5rem" }}>Registry Credentials</h2>
                <p style={{ color: "var(--text-muted)", fontSize: "0.9rem" }}>
                  Store credentials for Docker Hub, GHCR, and private registries used by OCI checks.
                </p>
              </div>

              <form onSubmit={handleRegistrySave} style={{ display: "grid", gap: "1rem" }}>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label>Server Address</label>
                  <input type="text" className="form-control" placeholder="docker.io" value={registryServer} onChange={(e) => setRegistryServer(e.target.value)} />
                </div>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label>Username</label>
                  <input type="text" className="form-control" placeholder="my-user" value={registryUsername} onChange={(e) => setRegistryUsername(e.target.value)} />
                </div>
                <div className="form-group" style={{ marginBottom: 0 }}>
                  <label>Password / Access Token</label>
                  <input type="password" className="form-control" placeholder="••••••••" value={registryPassword} onChange={(e) => setRegistryPassword(e.target.value)} />
                </div>
                <button type="submit" className="btn btn-primary" disabled={isRegistrySaving}>
                  {isRegistrySaving ? "Saving..." : "Save Registry Credential"}
                </button>
              </form>

              <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
                {registries.length === 0 ? (
                  <div style={{ color: "var(--text-muted)", fontSize: "0.9rem" }}>No registry credentials configured yet.</div>
                ) : registries.map((registry) => (
                  <div key={registry.id} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "0.9rem", background: "rgba(255,255,255,0.02)", borderRadius: "8px" }}>
                    <div>
                      <div style={{ fontWeight: 600 }}>{registry.server_address}</div>
                      <div style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>{registry.username} • password: {registry.password}</div>
                    </div>
                    <button className="btn btn-secondary" onClick={() => handleDeleteRegistry(registry.id)}>Delete</button>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="glass-card" style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
              <div>
                <h2 style={{ fontSize: "1.2rem", marginBottom: "0.5rem" }}>GitOps Git Integration</h2>
                <p style={{ color: "var(--text-muted)", fontSize: "0.9rem" }}>
                  Settings are loaded from the backend host environment variables.
                </p>
              </div>

              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1.5rem" }}>
                <div style={{ display: "flex", flexDirection: "column", gap: "0.6rem", fontSize: "0.9rem" }}>
                  <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--glass-border)", paddingBottom: "0.5rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>Status:</span>
                    <span style={{ fontWeight: "600", color: "var(--color-success)" }}>Active (Operational)</span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--glass-border)", paddingBottom: "0.5rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>GitOps Mode:</span>
                    <span>Commit Manifests updates</span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--glass-border)", paddingBottom: "0.5rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>Target Branch:</span>
                    <span style={{ fontFamily: "monospace" }}>main</span>
                  </div>
                </div>

                <div style={{ display: "flex", flexDirection: "column", gap: "0.6rem", fontSize: "0.9rem" }}>
                  <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--glass-border)", paddingBottom: "0.5rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>Remote Sync:</span>
                    <span>Auto Push enabled</span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--glass-border)", paddingBottom: "0.5rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>Authentication:</span>
                    <span>SSH Key (id_rsa)</span>
                  </div>
                  <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--glass-border)", paddingBottom: "0.5rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>SQLite Path:</span>
                    <span style={{ fontFamily: "monospace" }}>/app/data/db.sqlite</span>
                  </div>
                </div>
              </div>
            </div>
          )}

        </section>
      </main>

      {/* Footer */}
      <footer style={{ borderTop: "1px solid var(--glass-border)", padding: "1.5rem", textAlign: "center", color: "var(--text-muted)", fontSize: "0.85rem", background: "var(--bg-secondary)", marginTop: "2rem" }}>
        Container Updater Console &bull; Settings panel
      </footer>
    </div>
  );
}
