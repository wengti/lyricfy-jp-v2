// Name of the session cookie. Shared by proxy.ts (optimistic check) and lib/auth/require-session.ts.
export const SESSION_COOKIE = "lyricfy_session";

// Route handler that clears a rejected session cookie, then sends the user to log in.
export const SESSION_EXPIRED_PATH = "/session/expired";
