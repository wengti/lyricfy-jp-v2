// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";
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

const cookieJar = { set: vi.fn(), delete: vi.fn() };

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
	it("deletes the session cookie and returns to the login page", async () => {
		await expect(logout()).rejects.toThrow("NEXT_REDIRECT:/");

		expect(cookieJar.delete).toHaveBeenCalledWith(SESSION_COOKIE);
	});
});
