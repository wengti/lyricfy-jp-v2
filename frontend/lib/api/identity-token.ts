import "server-only";

// On Cloud Run the backend is private: Cloud Run only lets a request through if it carries a
// Google-signed identity token for an account allowed to invoke the backend (the frontend's
// service account). The metadata server that Cloud Run runs next to every container issues those
// tokens. Locally there's neither the check nor the metadata server, so no token is sent.

const METADATA_IDENTITY_URL =
	"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity";

// Tokens last an hour; fetch a new one this long before the cached one expires
const REFRESH_MARGIN_MS = 5 * 60 * 1000;

let cached: { audience: string; token: string; expiresAt: number } | undefined;

/** True on Cloud Run, which sets K_SERVICE in every container. */
export function onCloudRun(): boolean {
	return Boolean(process.env.K_SERVICE);
}

/** Returns an identity token for the given audience (the backend's URL), reused until near expiry. */
export async function getIdentityToken(audience: string): Promise<string> {
	if (cached && cached.audience === audience && Date.now() < cached.expiresAt - REFRESH_MARGIN_MS) {
		return cached.token;
	}

	const res = await fetch(`${METADATA_IDENTITY_URL}?audience=${encodeURIComponent(audience)}`, {
		headers: { "Metadata-Flavor": "Google" },
		cache: "no-store",
	});
	if (!res.ok) {
		throw new Error(`Metadata server returned ${res.status} for an identity token`);
	}

	const token = await res.text();
	cached = { audience, token, expiresAt: expiryOf(token) };
	return token;
}

/** The token's exp claim in milliseconds. A JWT's payload is base64url-encoded JSON. */
function expiryOf(token: string): number {
	const payload = JSON.parse(Buffer.from(token.split(".")[1], "base64url").toString()) as { exp: number };
	return payload.exp * 1000;
}

/** Forgets the cached token. For tests. */
export function resetIdentityTokenCache(): void {
	cached = undefined;
}
