declare global {
  interface Window {
    __ENV__?: {
      NEXT_PUBLIC_BACKEND_URL?: string;
    };
  }
}

export function getBackendUrl(): string {
  if (typeof window !== "undefined") {
    if (window.__ENV__ && window.__ENV__.NEXT_PUBLIC_BACKEND_URL) {
      return window.__ENV__.NEXT_PUBLIC_BACKEND_URL;
    }
  }

  const envUrl = process.env.NEXT_PUBLIC_BACKEND_URL;
  if (envUrl && envUrl !== "http://localhost:8080") {
    return envUrl;
  }

  if (typeof window !== "undefined") {
    const protocol = window.location.protocol;
    const hostname = window.location.hostname;
    return `${protocol}//${hostname}:8080`;
  }

  return "http://localhost:8080";
}

export function getWebSocketUrl(): string {
  const backendUrl = getBackendUrl();
  return backendUrl.replace(/^http/, "ws") + "/api/ws";
}
