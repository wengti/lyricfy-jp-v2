// @vitest-environment node
import { describe, expect, it, vi } from "vitest";
import { cookies } from "next/headers";
import { requireSession } from "./require-session";
import { BackendUnavailableError } from "@/lib/api/client";
import { SESSION_COOKIE, SESSION_EXPIRED_PATH } from "@/lib/session/cookie";
import { jsonResponse, mockBackend, TEST_API_URL } from "@/lib/test-utils/backend";

// `server-only` throws when imported outside a Next.js server build; it's a no-op here
vi.mock("server-only", () => ({}));
// Outside a real request there are no cookies to read, so the test supplies them
vi.mock("next/headers", () => ({ cookies: vi.fn() }));
// vi.mock is hoisted above the imports, so the helper is loaded inside the factory
vi.mock("next/navigation", async () => {
	const { throwingRedirect } = await import("@/lib/test-utils/backend");
	return { redirect: vi.fn(throwingRedirect) };
});

function withCookies(jar: Record<string, string>) {
	vi.mocked(cookies).mockResolvedValue({
		get: (name: string) => (name in jar ? { name, value: jar[name] } : undefined),
	} as unknown as Awaited<ReturnType<typeof cookies>>);
}

describe("requireSession", () => {
	it("sends visitors without a session cookie to the login page", async () => {
		const fetchMock = mockBackend();
		withCookies({});

		await expect(requireSession()).rejects.toThrow("NEXT_REDIRECT:/");
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it("verifies the token with the backend and returns the user", async () => {
		const fetchMock = mockBackend();
		fetchMock.mockResolvedValue(jsonResponse(200, { data: { email: "me@example.com" } }));
		withCookies({ [SESSION_COOKIE]: "session-token" });

		await expect(requireSession()).resolves.toEqual({ email: "me@example.com" });

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe(`${TEST_API_URL}/api/auth/me`);
		expect(init?.headers).toMatchObject({ Authorization: "Bearer session-token" });
	});

	it("sends rejected tokens to the route that clears the cookie", async () => {
		mockBackend().mockResolvedValue(jsonResponse(401, { error: "invalid or expired session" }));
		withCookies({ [SESSION_COOKIE]: "expired-token" });

		await expect(requireSession()).rejects.toThrow(`NEXT_REDIRECT:${SESSION_EXPIRED_PATH}`);
	});

	it("doesn't log you out when the backend is down", async () => {
		mockBackend().mockRejectedValue(new TypeError("fetch failed"));
		withCookies({ [SESSION_COOKIE]: "session-token" });

		await expect(requireSession()).rejects.toBeInstanceOf(BackendUnavailableError);
	});

	it("fails loudly on unexpected backend errors", async () => {
		mockBackend().mockResolvedValue(jsonResponse(500, { error: "Internal Server Error" }));
		withCookies({ [SESSION_COOKIE]: "session-token" });

		await expect(requireSession()).rejects.toThrow("Session check failed: 500");
	});
});
