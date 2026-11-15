// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cookies } from "next/headers";
import { login, logout } from "./auth";
import { SESSION_COOKIE } from "@/lib/session/cookie";
import { jsonResponse, mockBackend, TEST_API_URL } from "@/lib/test-utils/backend";

vi.mock("server-only", () => ({}));
vi.mock("next/headers", () => ({ cookies: vi.fn() }));
// vi.mock is hoisted above the imports, so the helper is loaded inside the factory
vi.mock("next/navigation", async () => {
	const { throwingRedirect } = await import("@/lib/test-utils/backend");
	return { redirect: vi.fn(throwingRedirect) };
});

const cookieJar = { get: vi.fn(), set: vi.fn(), delete: vi.fn() };

function form(fields: Record<string, string>) {
	const data = new FormData();
	for (const [key, value] of Object.entries(fields)) data.set(key, value);
	return data;
}

const NO_ERROR = { error: null };
const VALID = { email: "me@example.com", password: "secret" };

beforeEach(() => {
	vi.mocked(cookies).mockResolvedValue(cookieJar as unknown as Awaited<ReturnType<typeof cookies>>);
});

describe("login", () => {
	it.each([
		["the email is missing", { email: "", password: "secret" }],
		["the password is missing", { email: "me@example.com", password: "" }],
		["the email is only whitespace", { email: "   ", password: "secret" }],
	])("asks for both fields when %s, without calling the backend", async (_, fields) => {
		const fetchMock = mockBackend();

		const state = await login(NO_ERROR, form(fields));

		expect(state.error).toBe("Please enter your email and password.");
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it("sends the trimmed email and the password to the backend", async () => {
		const fetchMock = mockBackend();
		fetchMock.mockResolvedValue(jsonResponse(401, { error: "invalid email or password" }));

		await login(NO_ERROR, form({ email: "  me@example.com ", password: "secret" }));

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe(`${TEST_API_URL}/api/auth/login`);
		expect(init?.method).toBe("POST");
		expect(JSON.parse(init?.body as string)).toEqual({ email: "me@example.com", password: "secret" });
	});

	it("stores the session in an httpOnly cookie and goes to the dictionary", async () => {
		mockBackend().mockResolvedValue(
			jsonResponse(200, { data: { token: "session-token", expiresAt: "2026-11-25T12:00:00Z" } }),
		);

		await expect(login(NO_ERROR, form(VALID))).rejects.toThrow("NEXT_REDIRECT:/dictionary");

		expect(cookieJar.set).toHaveBeenCalledWith(SESSION_COOKIE, "session-token", {
			httpOnly: true,
			secure: false, // tests don't run in production mode
			sameSite: "lax",
			path: "/",
			expires: new Date("2026-11-25T12:00:00Z"),
		});
	});

	it("reports wrong credentials without setting a cookie", async () => {
		mockBackend().mockResolvedValue(jsonResponse(401, { error: "invalid email or password" }));

		const state = await login(NO_ERROR, form(VALID));

		expect(state.error).toBe("Invalid email or password.");
		expect(cookieJar.set).not.toHaveBeenCalled();
	});

	it("shows a generic message for unexpected backend errors", async () => {
		mockBackend().mockResolvedValue(jsonResponse(500, { error: "Internal Server Error" }));

		const state = await login(NO_ERROR, form(VALID));

		expect(state.error).toBe("Something went wrong. Please try again.");
	});

	it("says when the backend can't be reached", async () => {
		mockBackend().mockRejectedValue(new TypeError("fetch failed"));

		const state = await login(NO_ERROR, form(VALID));

		expect(state.error).toBe("Can't reach the server. Please try again in a moment.");
		expect(cookieJar.set).not.toHaveBeenCalled();
	});
});

describe("logout", () => {
	// Puts back console.error after the tests that silence it
	afterEach(() => {
		vi.restoreAllMocks();
	});

	function withSessionCookie(token: string) {
		cookieJar.get.mockImplementation((name: string) => (name === SESSION_COOKIE ? { name, value: token } : undefined));
	}

	it("ends the session on the backend, then deletes the cookie and returns to the login page", async () => {
		withSessionCookie("session-token");
		const fetchMock = mockBackend();
		fetchMock.mockResolvedValue(jsonResponse(200, { data: null }));

		await expect(logout()).rejects.toThrow("NEXT_REDIRECT:/");

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe(`${TEST_API_URL}/api/auth/logout`);
		expect(init?.method).toBe("POST");
		expect((init?.headers as Record<string, string>).Authorization).toBe("Bearer session-token");
		expect(cookieJar.delete).toHaveBeenCalledWith(SESSION_COOKIE);
	});

	it("still logs out on this device when the backend can't be reached", async () => {
		withSessionCookie("session-token");
		mockBackend().mockRejectedValue(new TypeError("fetch failed"));
		const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

		await expect(logout()).rejects.toThrow("NEXT_REDIRECT:/");

		expect(cookieJar.delete).toHaveBeenCalledWith(SESSION_COOKIE);
		expect(consoleError).toHaveBeenCalled();
	});

	it("still logs out on this device when the frontend is misconfigured", async () => {
		withSessionCookie("session-token");
		const fetchMock = mockBackend();
		vi.stubEnv("API_URL", ""); // apiFetch throws before calling the backend
		const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

		await expect(logout()).rejects.toThrow("NEXT_REDIRECT:/");

		expect(fetchMock).not.toHaveBeenCalled();
		expect(cookieJar.delete).toHaveBeenCalledWith(SESSION_COOKIE);
		expect(consoleError).toHaveBeenCalled();
	});

	it("still logs out on this device when the backend returns an error", async () => {
		withSessionCookie("session-token");
		mockBackend().mockResolvedValue(jsonResponse(500, { error: "Internal Server Error" }));
		const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

		await expect(logout()).rejects.toThrow("NEXT_REDIRECT:/");

		expect(cookieJar.delete).toHaveBeenCalledWith(SESSION_COOKIE);
		expect(consoleError).toHaveBeenCalledWith("Logout failed on the backend: 500 Internal Server Error");
	});

	it("skips the backend when there's no session cookie", async () => {
		cookieJar.get.mockReturnValue(undefined);
		const fetchMock = mockBackend();

		await expect(logout()).rejects.toThrow("NEXT_REDIRECT:/");

		expect(fetchMock).not.toHaveBeenCalled();
		expect(cookieJar.delete).toHaveBeenCalledWith(SESSION_COOKIE);
	});
});
