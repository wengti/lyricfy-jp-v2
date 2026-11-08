import "server-only";
import { getIdentityToken, onCloudRun } from "./identity-token";

// Server-side client for the Go backend. Never import this into client components:
// it sends FRONTEND_API_KEY, which only the Next.js server may know.

export type ApiResult<T> = { ok: true; data: T } | { ok: false; status: number; error: string };

/**
 * Thrown when the backend can't be reached at all (down, wrong URL, network error, or on Cloud Run
 * no identity token to get past it).
 */
export class BackendUnavailableError extends Error {
	constructor(cause: unknown) {
		super("Backend is unavailable", { cause });
		this.name = "BackendUnavailableError";
	}
}

function env(name: "API_URL" | "FRONTEND_API_KEY"): string {
	const value = process.env[name];
	if (!value) {
		throw new Error(`${name} is not set. Copy frontend/.env.example to frontend/.env.local and fill it in.`);
	}
	return value;
}

/**
 * Calls the backend and unwraps its standard response shape ({"data": ...} or {"error": "..."}).
 * HTTP errors come back as { ok: false }; only an unreachable backend throws.
 */
export async function apiFetch<T>(
	path: string,
	{ method = "GET", body, token }: { method?: string; body?: unknown; token?: string } = {},
): Promise<ApiResult<T>> {
	const apiUrl = env("API_URL");
	const headers: Record<string, string> = { "X-Frontend-Key": env("FRONTEND_API_KEY") };
	if (body !== undefined) headers["Content-Type"] = "application/json";
	if (token) headers["Authorization"] = `Bearer ${token}`;

	let res: Response;
	try {
		// On Cloud Run, prove to Cloud Run that the caller is the frontend (see identity-token.ts).
		// It has its own header because Authorization carries the session token for the backend.
		if (onCloudRun()) headers["X-Serverless-Authorization"] = `Bearer ${await getIdentityToken(apiUrl)}`;

		res = await fetch(`${apiUrl}${path}`, {
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body),
			cache: "no-store",
		});
	} catch (err) {
		throw new BackendUnavailableError(err);
	}

	const json = (await res.json().catch(() => ({}))) as { data?: T; error?: string };
	if (res.ok) return { ok: true, data: json.data as T };
	return { ok: false, status: res.status, error: json.error ?? res.statusText };
}
