// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { requireSession } from "./auth";
import { SESSION_COOKIE } from "./session";

// `server-only` throws when imported outside a Next.js server build; it's a no-op here
vi.mock("server-only", () => ({}));
// Outside a real request there are no cookies to read, so the test supplies them.
// The real redirect() throws to stop rendering; the mock just records the call.
vi.mock("next/headers", () => ({ cookies: vi.fn() }));
vi.mock("next/navigation", () => ({ redirect: vi.fn() }));

function withCookies(jar: Record<string, string>) {
	vi.mocked(cookies).mockResolvedValue({
		get: (name: string) => (name in jar ? { name, value: jar[name] } : undefined),
	} as unknown as Awaited<ReturnType<typeof cookies>>);
}

describe("requireSession", () => {
	beforeEach(() => {
		vi.mocked(redirect).mockClear();
	});

	it("redirects to the login page when there is no session cookie", async () => {
		withCookies({});
		await requireSession();
		expect(redirect).toHaveBeenCalledWith("/");
	});

	it("redirects when the session cookie is empty", async () => {
		withCookies({ [SESSION_COOKIE]: "" });
		await requireSession();
		expect(redirect).toHaveBeenCalledWith("/");
	});

	it("lets the request through when a session cookie exists", async () => {
		withCookies({ [SESSION_COOKIE]: "any-token" });
		await requireSession();
		expect(redirect).not.toHaveBeenCalled();
	});
});
