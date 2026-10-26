import { NextResponse, type NextRequest } from "next/server";
import { SESSION_COOKIE } from "@/lib/session/cookie";

const PROTECTED = ["/dictionary"];

// Optimistic redirects based on the session cookie's presence only.
// The real check happens server-side in lib/auth/require-session.ts (requireSession).
export function proxy(request: NextRequest) {
	const { pathname } = request.nextUrl; // where the visitor is going to, i.e. /dictionary
	const hasSession = request.cookies.has(SESSION_COOKIE); // whether the request carries the cookies

	// visitor has no cookies and visting a PROTECTED route
	if (!hasSession && PROTECTED.some((p) => pathname.startsWith(p))) {
		return NextResponse.redirect(new URL("/", request.url)); // build the fullURL to redirect to the home page
	}

	// The login page is the landing page; skip it when already logged in
	if (hasSession && pathname === "/") {
		return NextResponse.redirect(new URL("/dictionary", request.url));
	}

	return NextResponse.next();
}

export const config = {
	matcher: ["/", "/dictionary/:path*"],
};
