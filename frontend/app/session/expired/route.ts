import { NextResponse, type NextRequest } from "next/server";
import { SESSION_COOKIE } from "@/lib/session/cookie";

// requireSession sends rejected sessions here: clear the cookie, then go to the login page.
export function GET(request: NextRequest) {
	const response = NextResponse.redirect(new URL("/", request.url));
	response.cookies.delete(SESSION_COOKIE);
	return response;
}
