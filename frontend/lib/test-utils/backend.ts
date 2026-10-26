import { vi } from "vitest";

// Shared helpers for tests that talk to the (mocked) backend.

export const TEST_API_URL = "http://backend.test";
export const TEST_FRONTEND_KEY = "test-frontend-key";

/** Points lib/api at a fake backend and replaces fetch; returns the fetch mock. */
export function mockBackend() {
	vi.stubEnv("API_URL", TEST_API_URL);
	vi.stubEnv("FRONTEND_API_KEY", TEST_FRONTEND_KEY);
	const fetchMock = vi.fn<typeof fetch>();
	vi.stubGlobal("fetch", fetchMock);
	return fetchMock;
}

/** A backend response in its standard shape. */
export function jsonResponse(status: number, body: unknown) {
	return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

/** Mimics Next's redirect(), which throws to stop rendering or the action. */
export function throwingRedirect(url: string): never {
	throw new Error(`NEXT_REDIRECT:${url}`);
}
