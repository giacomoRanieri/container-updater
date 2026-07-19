const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface SessionInfo {
  authenticated: boolean;
  user?: User;
}

export async function fetchSession(): Promise<SessionInfo> {
  try {
    const res = await fetch(`${BACKEND_URL}/api/auth/session`, {
      headers: {
        "Accept": "application/json",
      },
      credentials: "include",
    });

    if (res.status === 401) {
      return { authenticated: false };
    }

    if (!res.ok) {
      throw new Error(`Session request failed: ${res.status}`);
    }

    return await res.json();
  } catch (error) {
    console.error("fetchSession error:", error);
    return { authenticated: false };
  }
}

export function redirectToLogin(): void {
  window.location.href = `${BACKEND_URL}/api/auth/login`;
}

export function logout(): void {
  // Clearing the session cookie requires making a call to backend or just expiring on client
  // In our OIDC design, backend handles the HTTP-only cookie.
  // For standard logout, we can also clear client-side states and redirect to login
  redirectToLogin();
}
