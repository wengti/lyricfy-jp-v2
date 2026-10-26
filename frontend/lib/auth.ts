import "server-only";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { SESSION_COOKIE } from "./session";

/**
 * Server-side gate for protected pages. Redirects to the login page when there is no session.
 *
 * TODO(backend): only checks that the cookie exists. Once the Go backend issues sessions,
 * verify the token with the backend here instead of trusting its presence.
 */
export async function requireSession(): Promise<void> {
	const session = (await cookies()).get(SESSION_COOKIE)?.value;
	if (!session) {
		redirect("/");
	}
}
