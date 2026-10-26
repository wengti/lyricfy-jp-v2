"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { SESSION_COOKIE } from "@/lib/session";

export interface LoginState {
	error: string | null;
}

// TODO(backend): send the credentials to the Go backend, which checks them against the
// config record and returns a session token; set it as an httpOnly SESSION_COOKIE, then
// redirect("/dictionary").
export async function login(_prev: LoginState, formData: FormData): Promise<LoginState> {
	const email = String(formData.get("email") ?? "").trim();
	const password = String(formData.get("password") ?? "");
	if (!email || !password) {
		return { error: "Please enter your email and password." };
	}
	return { error: "Login isn't connected to the backend yet." };
}

// TODO(backend): also invalidate the session on the Go backend.
export async function logout(): Promise<void> {
	(await cookies()).delete(SESSION_COOKIE);
	redirect("/");
}
