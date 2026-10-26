// @vitest-environment node
import { describe, expect, it, vi } from "vitest";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { login, logout } from "./auth";
import { SESSION_COOKIE } from "@/lib/session";

vi.mock("next/headers", () => ({ cookies: vi.fn() }));
vi.mock("next/navigation", () => ({ redirect: vi.fn() }));

function form(fields: Record<string, string>) {
	const data = new FormData();
	for (const [key, value] of Object.entries(fields)) data.set(key, value);
	return data;
}

const NO_ERROR = { error: null };

describe("login", () => {
	it("asks for both fields when the email is missing", async () => {
		const state = await login(NO_ERROR, form({ email: "", password: "secret" }));
		expect(state.error).toBe("Please enter your email and password.");
	});

	it("asks for both fields when the password is missing", async () => {
		const state = await login(NO_ERROR, form({ email: "me@example.com", password: "" }));
		expect(state.error).toBe("Please enter your email and password.");
	});

	it("treats a whitespace-only email as missing", async () => {
		const state = await login(NO_ERROR, form({ email: "   ", password: "secret" }));
		expect(state.error).toBe("Please enter your email and password.");
	});

	// Placeholder behaviour until the Go backend checks credentials. Update this test then.
	it("reports that the backend isn't connected yet", async () => {
		const state = await login(NO_ERROR, form({ email: "me@example.com", password: "secret" }));
		expect(state.error).toBe("Login isn't connected to the backend yet.");
	});
});

describe("logout", () => {
	it("deletes the session cookie and returns to the login page", async () => {
		const deleteCookie = vi.fn();
		vi.mocked(cookies).mockResolvedValue({
			delete: deleteCookie,
		} as unknown as Awaited<ReturnType<typeof cookies>>);

		await logout();

		expect(deleteCookie).toHaveBeenCalledWith(SESSION_COOKIE);
		expect(redirect).toHaveBeenCalledWith("/");
	});
});
