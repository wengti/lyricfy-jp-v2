// @vitest-environment node
import { describe, expect, it } from "vitest";
import { NextRequest } from "next/server";
import { getRedirectUrl } from "next/experimental/testing/server";
import { GET } from "./route";
import { SESSION_COOKIE } from "@/lib/session/cookie";

describe("GET /session/expired", () => {
	it("clears the session cookie and sends the user to the login page", () => {
		const req = new NextRequest("http://localhost:3000/session/expired");
		req.cookies.set(SESSION_COOKIE, "expired-token");

		const res = GET(req);

		expect(getRedirectUrl(res)).toBe("http://localhost:3000/");
		const cleared = res.cookies.get(SESSION_COOKIE);
		expect(cleared?.value).toBe("");
		expect(cleared?.expires).toEqual(new Date(0));
	});
});
