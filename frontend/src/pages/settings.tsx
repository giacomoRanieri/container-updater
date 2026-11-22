import React, { useState, useEffect, useCallback } from "react";
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
  const [isLoading, setIsLoading] = useState(true);
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

  const showMsg = useCallback((text: string, type: "success" | "error") => {
    setMessage({ text, type });
    setTimeout(() => setMessage(null), 5000);
  }, []);

  // Load notification settings
  const loadSettings = useCallback(async () => {
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
  }, [showMsg]);

  useEffect(() => {
    let active = true;
    fetchNotifications()
      .then((list) => {
        if (active) setNotifications(list);
      })
      .catch((err) => {
        console.error("Failed to load notification settings", err);
        if (active) showMsg("Failed to retrieve notification configurations", "error");
      })
      .finally(() => {
        if (active) setIsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [showMsg]);

  const loadRegistries = useCallback(async () => {
    try {
      const list = await fetchRegistries();
      setRegistries(list);
    } catch (err) {
      console.error("Failed to load registry credentials", err);
      showMsg("Failed to retrieve registry credentials", "error");
    }
  }, [showMsg]);

  useEffect(() => {
    if (activeTab !== "registries") return;
    let active = true;
    fetchRegistries()
      .then((list) => {
        if (active) setRegistries(list);
      })
      .catch((err) => {
        console.error("Failed to load registry credentials", err);
        if (active) showMsg("Failed to retrieve registry credentials", "error");
      });
    return () => {
      active = false;
    };
  }, [activeTab, showMsg]);

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
          <Link href="/" style={{ display: "flex", alignItems: "center", gap: "0.75rem", textDecoration: "none", color: "inherit" }}>
            <div style={{ background: "var(--accent-gradient)", width: "32px", height: "32px", borderRadius: "8px", display: "inline-flex", alignItems: "center", justifyContent: "center", color: "white" }}>
              <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
              </svg>
            </div>
            <span style={{ fontSize: "1.2rem", fontWeight: "600" }}>
              Container <span className="text-gradient">Updater</span>
            </span>
          </Link>

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
      <main className="main-content settings-layout">
        
        {/* Settings Sidebar Tabs */}
        <aside className="glass-card settings-sidebar">
          <button
            type="button"
            className={`settings-nav-btn${activeTab === "notifications" ? " active" : ""}`}
            onClick={() => setActiveTab("notifications")}
          >
            <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0"></path>
            </svg>
            <span>Apprise Notifications</span>
          </button>
          <button
            type="button"
            className={`settings-nav-btn${activeTab === "registries" ? " active" : ""}`}
            onClick={() => setActiveTab("registries")}
          >
            <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <rect x="3" y="4" width="18" height="16" rx="3"></rect>
              <path d="M7 8h10M7 12h10M7 16h6"></path>
            </svg>
            <span>Registries</span>
          </button>
          <button
            type="button"
            className={`settings-nav-btn${activeTab === "system" ? " active" : ""}`}
            onClick={() => setActiveTab("system")}
          >
            <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <rect x="2" y="2" width="20" height="20" rx="5" ry="5"></rect>
              <path d="M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37zM17.5 6.5h.01"></path>
            </svg>
            <span>GitOps &amp; System Status</span>
          </button>
        </aside>

        {/* Settings Tab Content */}
        <section className="settings-content">
          
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
            <div className="settings-split-grid">
              {/* Form card */}
              <div className={`glass-card settings-form-card${editingId ? " is-editing" : ""}`}>
                <div className="settings-card-header">
                  <div>
                    <h2 className="settings-card-title">
                      {editingId ? "Modify Notification Endpoint" : "Add Notification Endpoint"}
                    </h2>
                    <p className="settings-card-subtitle">
                      Send rollout and update alerts through Apprise-compatible endpoints.
                    </p>
                  </div>
                  {editingId && <span className="settings-editing-badge">Editing</span>}
                </div>
                
                <form onSubmit={handleSave} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label htmlFor="notification-name">Configuration Name</label>
                    <input
                      id="notification-name"
                      type="text"
                      className="form-control"
                      placeholder="e.g. Admin Telegram Chat"
                      value={formName}
                      onChange={(e) => setFormName(e.target.value)}
                    />
                  </div>

                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label htmlFor="notification-url">Apprise Connection URL</label>
                    <input
                      id="notification-url"
                      type="text"
                      className="form-control"
                      placeholder="tgram://botToken/chatID"
                      value={formUrl}
                      onChange={(e) => setFormUrl(e.target.value)}
                    />
                    <small className="form-hint">
                      Apprise format examples: <code className="inline-code">tgram://bottoken/chatid</code> or <code className="inline-code">pushed://appkey</code>
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
                      style={{ flexGrow: 1, whiteSpace: "nowrap" }}
                    >
                      {isSaving ? "Saving..." : editingId ? "Update Configuration" : "Save Configuration"}
                    </button>
                    {editingId && (
                      <button
                        type="button"
                        className="btn btn-secondary"
                        onClick={handleCancelEdit}
                        style={{ whiteSpace: "nowrap" }}
                      >
                        Cancel
                      </button>
                    )}
                  </div>
                </form>
              </div>

              {/* Endpoints List card */}
              <div className="glass-card">
                <div className="settings-card-header">
                  <div>
                    <h2 className="settings-card-title">Configured Channels</h2>
                    <p className="settings-card-subtitle">
                      Active and paused endpoints notified during scans and updates.
                    </p>
                  </div>
                  <span className="settings-count-badge">
                    {notifications.length} {notifications.length === 1 ? "channel" : "channels"}
                  </span>
                </div>

                {isLoading ? (
                  <div style={{ padding: "2rem", textAlign: "center" }}>
                    <svg className="spin" width="24" height="24" fill="none" stroke="var(--accent-indigo)" strokeWidth="2" viewBox="0 0 24 24">
                      <circle cx="12" cy="12" r="10" strokeDasharray="15 5"></circle>
                    </svg>
                  </div>
                ) : notifications.length === 0 ? (
                  <div className="settings-empty-state">
                    No notification channels registered.
                  </div>
                ) : (
                  <div className="settings-list">
                    {notifications.map((n) => (
                      <div
                        key={n.id}
                        className={`settings-list-item${editingId === n.id ? " is-editing" : ""}`}
                      >
                        <div className="settings-item-body">
                          <div className="settings-item-title-row">
                            <h4 className="settings-item-title" title={n.name}>
                              {n.name}
                            </h4>
                            <span className={`channel-status-pill ${n.is_enabled ? "is-enabled" : "is-disabled"}`}>
                              {n.is_enabled ? "Enabled" : "Disabled"}
                            </span>
                          </div>
                          <code className="settings-item-url" title={n.apprise_url}>
                            {n.apprise_url}
                          </code>
                        </div>
                        <div className="settings-item-actions">
                          <button
                            type="button"
                            className="btn btn-secondary btn-sm"
                            onClick={() => handleEdit(n)}
                          >
                            {editingId === n.id ? "Editing" : "Modify"}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : activeTab === "registries" ? (
            <div className="settings-split-grid">
              <div className="glass-card">
                <div className="settings-card-header">
                  <div>
                    <h2 className="settings-card-title">Add Registry Credential</h2>
                    <p className="settings-card-subtitle">
                      Store credentials for Docker Hub, GHCR, and private registries used by OCI checks.
                    </p>
                  </div>
                </div>

                <form onSubmit={handleRegistrySave} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label htmlFor="registry-server">Server Address</label>
                    <input
                      id="registry-server"
                      type="text"
                      className="form-control"
                      placeholder="docker.io"
                      value={registryServer}
                      onChange={(e) => setRegistryServer(e.target.value)}
                    />
                  </div>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label htmlFor="registry-username">Username</label>
                    <input
                      id="registry-username"
                      type="text"
                      className="form-control"
                      placeholder="my-user"
                      value={registryUsername}
                      onChange={(e) => setRegistryUsername(e.target.value)}
                    />
                  </div>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label htmlFor="registry-password">Password / Access Token</label>
                    <input
                      id="registry-password"
                      type="password"
                      className="form-control"
                      placeholder="••••••••"
                      value={registryPassword}
                      onChange={(e) => setRegistryPassword(e.target.value)}
                    />
                  </div>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={isRegistrySaving}
                    style={{ width: "100%", marginTop: "0.5rem", whiteSpace: "nowrap" }}
                  >
                    {isRegistrySaving ? "Saving..." : "Save Registry Credential"}
                  </button>
                </form>
              </div>

              <div className="glass-card">
                <div className="settings-card-header">
                  <div>
                    <h2 className="settings-card-title">Configured Registries</h2>
                    <p className="settings-card-subtitle">
                      Credentials applied automatically when polling private images.
                    </p>
                  </div>
                  <span className="settings-count-badge">
                    {registries.length} {registries.length === 1 ? "registry" : "registries"}
                  </span>
                </div>

                {registries.length === 0 ? (
                  <div className="settings-empty-state">
                    No registry credentials configured yet.
                  </div>
                ) : (
                  <div className="settings-list">
                    {registries.map((registry) => (
                      <div key={registry.id} className="settings-list-item">
                        <div className="settings-item-body">
                          <h4 className="settings-item-title" title={registry.server_address}>
                            {registry.server_address}
                          </h4>
                          <div className="settings-item-meta" title={`${registry.username} • ${registry.password}`}>
                            {registry.username} &bull; {registry.password}
                          </div>
                        </div>
                        <div className="settings-item-actions">
                          <button
                            type="button"
                            className="btn btn-secondary btn-sm"
                            onClick={() => handleDeleteRegistry(registry.id)}
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="glass-card">
              <div className="settings-card-header">
                <div>
                  <h2 className="settings-card-title">GitOps Git Integration</h2>
                  <p className="settings-card-subtitle">
                    Settings are loaded from the backend host environment variables.
                  </p>
                </div>
              </div>

              <div className="settings-status-grid">
                <div className="settings-kv-list">
                  <div className="settings-kv-row">
                    <span style={{ color: "var(--text-secondary)" }}>Status:</span>
                    <span className="settings-kv-value" style={{ fontWeight: "600", color: "var(--color-success)" }}>
                      Active (Operational)
                    </span>
                  </div>
                  <div className="settings-kv-row">
                    <span style={{ color: "var(--text-secondary)" }}>GitOps Mode:</span>
                    <span className="settings-kv-value">Commit Manifests updates</span>
                  </div>
                  <div className="settings-kv-row">
                    <span style={{ color: "var(--text-secondary)" }}>Target Branch:</span>
                    <span className="settings-kv-value" style={{ fontFamily: "monospace" }}>main</span>
                  </div>
                </div>

                <div className="settings-kv-list">
                  <div className="settings-kv-row">
                    <span style={{ color: "var(--text-secondary)" }}>Remote Sync:</span>
                    <span className="settings-kv-value">Auto Push enabled</span>
                  </div>
                  <div className="settings-kv-row">
                    <span style={{ color: "var(--text-secondary)" }}>Authentication:</span>
                    <span className="settings-kv-value">SSH Key (id_rsa)</span>
                  </div>
                  <div className="settings-kv-row">
                    <span style={{ color: "var(--text-secondary)" }}>SQLite Path:</span>
                    <span className="settings-kv-value" style={{ fontFamily: "monospace" }}>/app/data/db.sqlite</span>
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
