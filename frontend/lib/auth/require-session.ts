import "server-only";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { apiFetch } from "@/lib/api/client";
import { SESSION_COOKIE, SESSION_EXPIRED_PATH } from "@/lib/session/cookie";

export interface User {
	email: string;
}

/**
 * Server-side gate for protected pages: verifies the session with the backend and returns
 * the logged-in user.
 *
 * - no session cookie → login page
 * - backend rejects the token (expired, forged) → SESSION_EXPIRED_PATH, which clears the
 *   cookie; pages can't delete cookies themselves, and keeping it would make proxy.ts
 *   bounce between the login page and here
 * - backend unreachable → throws, so a backend outage doesn't log you out
 */
export async function requireSession(): Promise<User> {
	const token = (await cookies()).get(SESSION_COOKIE)?.value;
	if (!token) {
		redirect("/");
	}

	const result = await apiFetch<User>("/api/auth/me", { token });
	if (!result.ok) {
		if (result.status === 401) {
			redirect(SESSION_EXPIRED_PATH);
		}
		throw new Error(`Session check failed: ${result.status} ${result.error}`);
	}
	return result.data;
}
