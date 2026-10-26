// @vitest-environment node
import { describe, expect, it } from "vitest";
import { NextRequest } from "next/server";
// The docs call the matcher helper `unstable_doesProxyMatch`; this Next.js version ships it
// under its pre-rename name.
import { getRedirectUrl, unstable_doesMiddlewareMatch } from "next/experimental/testing/server";
import { config, proxy } from "./proxy";
import { SESSION_COOKIE } from "@/lib/session/cookie";

const BASE = "http://localhost:3000";

function request(path: string, { loggedIn }: { loggedIn: boolean }) {
	const req = new NextRequest(`${BASE}${path}`);
	if (loggedIn) req.cookies.set(SESSION_COOKIE, "any-token");
	return req;
}

describe("proxy matcher", () => {
	it.each(["/", "/dictionary", "/dictionary/some-word"])("runs for %s", (url) => {
		expect(unstable_doesMiddlewareMatch({ config, url })).toBe(true); // check if the middleware (by config) will run
	});

	it.each(["/icon.png", "/_next/static/chunk.js", "/some-other-page"])("skips %s", (url) => {
		expect(unstable_doesMiddlewareMatch({ config, url })).toBe(false); // check if the middleware (by config) will run
	});
});

describe("proxy redirects", () => {
	describe("when logged out", () => {
		it("sends /dictionary to the login page", () => {
			const res = proxy(request("/dictionary", { loggedIn: false }));
			expect(getRedirectUrl(res)).toBe(`${BASE}/`);
		});

		it("sends nested dictionary pages to the login page", () => {
			const res = proxy(request("/dictionary/some-word", { loggedIn: false }));
			expect(getRedirectUrl(res)).toBe(`${BASE}/`);
		});

		it("shows the login page", () => {
			const res = proxy(request("/", { loggedIn: false }));
			expect(getRedirectUrl(res)).toBeNull();
		});
	});

	describe("when logged in", () => {
		it("skips the login page and goes to /dictionary", () => {
			const res = proxy(request("/", { loggedIn: true }));
			expect(getRedirectUrl(res)).toBe(`${BASE}/dictionary`);
		});

		it("lets /dictionary through", () => {
			const res = proxy(request("/dictionary", { loggedIn: true }));
			expect(getRedirectUrl(res)).toBeNull();
		});
	});
});
