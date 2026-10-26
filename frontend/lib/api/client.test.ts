// @vitest-environment node
import { describe, expect, it, vi } from "vitest";
import { apiFetch, BackendUnavailableError } from "./client";
import { jsonResponse, mockBackend, TEST_API_URL, TEST_FRONTEND_KEY } from "@/lib/test-utils/backend";

vi.mock("server-only", () => ({}));

describe("apiFetch", () => {
	it("calls the backend with the frontend key, session token and JSON body", async () => {
		const fetchMock = mockBackend();
		fetchMock.mockResolvedValue(jsonResponse(200, { data: {} }));

		await apiFetch("/api/things", { method: "POST", body: { a: 1 }, token: "tok" });

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe(`${TEST_API_URL}/api/things`);
		expect(init?.method).toBe("POST");
		expect(init?.body).toBe(JSON.stringify({ a: 1 }));
		expect(init?.headers).toMatchObject({
			"X-Frontend-Key": TEST_FRONTEND_KEY,
			Authorization: "Bearer tok",
			"Content-Type": "application/json",
		});
	});

	it("omits the token and body when not given", async () => {
		const fetchMock = mockBackend();
		fetchMock.mockResolvedValue(jsonResponse(200, { data: {} }));

		await apiFetch("/api/things");

		const init = fetchMock.mock.calls[0][1];
		expect(init?.method).toBe("GET");
		expect(init?.body).toBeUndefined();
		expect(init?.headers).not.toHaveProperty("Authorization");
	});

	it("unwraps the data of a successful response", async () => {
		mockBackend().mockResolvedValue(jsonResponse(200, { data: { email: "me@example.com" } }));

		await expect(apiFetch("/api/auth/me")).resolves.toEqual({ ok: true, data: { email: "me@example.com" } });
	});

	it("returns the status and error message of a failed response", async () => {
		mockBackend().mockResolvedValue(jsonResponse(401, { error: "not logged in" }));

		await expect(apiFetch("/api/auth/me")).resolves.toEqual({ ok: false, status: 401, error: "not logged in" });
	});

	it("throws BackendUnavailableError when the backend can't be reached", async () => {
		mockBackend().mockRejectedValue(new TypeError("fetch failed"));

		await expect(apiFetch("/api/auth/me")).rejects.toBeInstanceOf(BackendUnavailableError);
	});

	it("explains how to fix a missing env var", async () => {
		mockBackend();
		vi.stubEnv("FRONTEND_API_KEY", "");

		await expect(apiFetch("/api/auth/me")).rejects.toThrow("FRONTEND_API_KEY is not set");
	});
});
