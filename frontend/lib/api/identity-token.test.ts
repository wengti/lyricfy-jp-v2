// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getIdentityToken, onCloudRun, resetIdentityTokenCache } from "./identity-token";

vi.mock("server-only", () => ({}));

const AUDIENCE = "https://backend.example.run.app";
const NOW = new Date("2026-01-01T00:00:00Z");

/** A JWT-shaped token that expires `secondsLeft` after NOW. Only the payload's exp is read. */
function fakeToken(secondsLeft: number, name = "token"): string {
	const exp = NOW.getTime() / 1000 + secondsLeft;
	const payload = Buffer.from(JSON.stringify({ exp })).toString("base64url");
	return `header.${payload}.${name}`;
}

function mockMetadataServer() {
	const fetchMock = vi.fn<typeof fetch>();
	vi.stubGlobal("fetch", fetchMock);
	return fetchMock;
}

beforeEach(() => {
	resetIdentityTokenCache();
	vi.useFakeTimers({ now: NOW });
});

afterEach(() => {
	vi.useRealTimers();
});

describe("onCloudRun", () => {
	it("is true when Cloud Run's K_SERVICE is set", () => {
		vi.stubEnv("K_SERVICE", "lyricfy-frontend");
		expect(onCloudRun()).toBe(true);
	});

	it("is false locally", () => {
		vi.stubEnv("K_SERVICE", "");
		expect(onCloudRun()).toBe(false);
	});
});

describe("getIdentityToken", () => {
	it("asks the metadata server for a token for the audience", async () => {
		const fetchMock = mockMetadataServer();
		const token = fakeToken(3600);
		fetchMock.mockResolvedValue(new Response(token));

		await expect(getIdentityToken(AUDIENCE)).resolves.toBe(token);

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe(
			"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity" +
				`?audience=${encodeURIComponent(AUDIENCE)}`,
		);
		expect(init?.headers).toMatchObject({ "Metadata-Flavor": "Google" });
	});

	it("reuses the token until shortly before it expires", async () => {
		const fetchMock = mockMetadataServer();
		fetchMock
			.mockResolvedValueOnce(new Response(fakeToken(3600, "first")))
			.mockResolvedValueOnce(new Response(fakeToken(7200, "second")));

		const first = await getIdentityToken(AUDIENCE);
		vi.advanceTimersByTime(50 * 60 * 1000); // 10 minutes left: still fine
		expect(await getIdentityToken(AUDIENCE)).toBe(first);
		expect(fetchMock).toHaveBeenCalledTimes(1);

		vi.advanceTimersByTime(6 * 60 * 1000); // 4 minutes left: time for a new one
		expect(await getIdentityToken(AUDIENCE)).toMatch(/\.second$/);
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it("fetches a new token for a different audience", async () => {
		const fetchMock = mockMetadataServer();
		fetchMock.mockImplementation(async () => new Response(fakeToken(3600)));

		await getIdentityToken(AUDIENCE);
		await getIdentityToken("https://other.example.run.app");

		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it("throws when the metadata server refuses", async () => {
		mockMetadataServer().mockResolvedValue(new Response("nope", { status: 404 }));

		await expect(getIdentityToken(AUDIENCE)).rejects.toThrow("Metadata server returned 404");
	});
});
