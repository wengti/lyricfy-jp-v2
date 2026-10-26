"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { apiFetch, BackendUnavailableError } from "@/lib/api/client";
import { SESSION_COOKIE } from "@/lib/session/cookie";

export interface LoginState {
	error: string | null;
}

interface Session {
	token: string;
	expiresAt: string;
}

export async function login(_prev: LoginState, formData: FormData): Promise<LoginState> {
	const email = String(formData.get("email") ?? "").trim();
	const password = String(formData.get("password") ?? "");
	if (!email || !password) {
		return { error: "Please enter your email and password." };
	}

	let result;
	try {
		result = await apiFetch<Session>("/api/auth/login", { method: "POST", body: { email, password } });
	} catch (err) {
		if (err instanceof BackendUnavailableError) {
			return { error: "Can't reach the server. Please try again in a moment." };
		}
		throw err;
	}

	if (!result.ok) {
		return {
			error: result.status === 401 ? "Invalid email or password." : "Something went wrong. Please try again.",
		};
	}

	(await cookies()).set(SESSION_COOKIE, result.data.token, {
		httpOnly: true, // not readable from browser JavaScript
		secure: process.env.NODE_ENV === "production", // HTTPS only in production
		sameSite: "lax",
		path: "/",
		expires: new Date(result.data.expiresAt), // the cookie ends when the token does
	});
	// redirect() throws to stop the action, so it must stay outside the try/catch above
	redirect("/dictionary");
}

// Tokens are stateless, so logging out means forgetting this one. The backend can't revoke
// it yet; that comes with the database.
export async function logout(): Promise<void> {
	(await cookies()).delete(SESSION_COOKIE);
	redirect("/");
}
