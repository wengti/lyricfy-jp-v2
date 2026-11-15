# Session tokens: from JWT to database sessions

Decided 2026-11-08, implemented 2026-11-09. The code is in `backend/internal/service/auth/` and
`backend/internal/repository/session_repo.go`; the table is created by
`backend/internal/database/migrations/00001_create_sessions.sql`.

Lyricfy replaced its JWT session tokens with random opaque tokens stored in a Neon `sessions` table.
Logout now kills a token immediately, and every login still expires after 30 days (`session.ttl: 720h`).

With JWTs, logout only deleted the browser cookie, so a copied token stayed valid until it expired.

## How a JWT works

A JWT (JSON Web Token) is a small signed note: the server writes facts about the user into it, seals it
with a secret, and stores nothing. On each request it checks the seal and reads the facts back out. That
is what "stateless" means: no per-login record exists on the server.

The example below uses real values, generated with the example secret `my-very-long-secret-from-config-yml`
and the same claims our Go backend issues.

### 1. Login: the server builds the token

The browser posts the email and password to `POST /api/auth/login`. After bcrypt confirms the password,
the backend builds three parts.

**Header**, which names the signing algorithm:

```
{"alg":"HS256","typ":"JWT"}
  → base64url → eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
```

**Payload**, which holds the claims. The login happens on 2026-11-08 and the TTL is 30 days:

```
{"iss":"lyricfy-api","sub":"you@example.com","exp":1796688000,"iat":1794096000}
                                                    └ 2026-12-08    └ 2026-11-08
  → base64url → eyJpc3MiOiJseXJpY2Z5LWFwaSIsInN1YiI6InlvdUBleGFtcGxlLmNvbSIsImV4cCI6MTc5NjY4ODAwMCwiaWF0IjoxNzk0MDk2MDAwfQ
```

**Signature**, an HMAC over `header.payload` keyed with the secret:

```
HMAC-SHA256(secret, "eyJhbGci...IkpXVCJ9.eyJpc3Mi...MDAwfQ")
  → base64url → FnxrF9Vr5C7l2mlELxSubDThcaP18e-f3gZSbnRYI-Q
```

Joined with dots, that is the JWT. The server returns it with `expiresAt` and keeps no record of it:

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJseXJpY2Z5LWFwaSIsInN1YiI6InlvdUBleGFtcGxlLmNvbSIsImV4cCI6MTc5NjY4ODAwMCwiaWF0IjoxNzk0MDk2MDAwfQ.FnxrF9Vr5C7l2mlELxSubDThcaP18e-f3gZSbnRYI-Q
```

The frontend stores it in the `lyricfy_session` cookie, set to expire on 2026-12-08. The client never
builds or parses the token; it only carries it.

### 2. Each request: the server verifies it

The frontend server sends `Authorization: Bearer <token>`. The backend then:

1. Splits the token at the dots into header, payload and signature.
2. Recomputes the signature from the header and payload it received, using its secret.
3. Compares the result with the signature in the token. A match proves the payload is exactly what the
   server wrote.
4. Decodes the payload and checks the claims: `iss` must be `lyricfy-api`, and `exp` must be in the future.
5. Accepts the request as `sub`, here `you@example.com`.

No database is involved: only the secret and the clock.

### 3. Tampering fails

Anyone holding the token can read the payload, because base64 is not encryption. Nobody without the
secret can change it. Suppose an attacker edits `exp` to 2030-01-01 (`1893456000`):

| Attacker's attempt | Signature in the token | Server recomputes | Result |
| --- | --- | --- | --- |
| Keep the original signature | `FnxrF9Vr…` | `pOeoPMNi…` | 401, mismatch |
| Sign it with a guessed secret | `b6CL9O3K…` | `pOeoPMNi…` | 401, mismatch |

Only the real secret produces `pOeoPMNi…`. For the same reason, nothing secret should ever go in a JWT
payload.

### 4. Expiry and logout

After 2026-12-08 the signature still matches, but the `exp` check fails and the server returns 401.
Logout is the gap: the server stored nothing, so it has nothing to delete. A token copied before logout
keeps passing every check until `exp`.

## Why we are not keeping JWT

Revoking a JWT early means the server must remember something, and that memory removes the only advantage
a JWT has. Since Neon is coming anyway, we keep sessions there and drop the JWT.

### Revocation needs server-side state

The usual fix is a denylist. Each token gets a random ID (`jti`), and logout stores that ID somewhere
shared. Every request then checks the denylist before accepting the token.

Redis fits this well. Logout runs `SET revoked:<jti> 1 EX <seconds until exp>`, and Redis deletes the
entry by itself when the token would have expired anyway. So Redis can invalidate JWTs; the problem is
the cost:

- **Another service to run.** Memorystore needs VPC networking and costs roughly $35+/month while idle.
  Upstash avoids that but is still a separate account, secret and failure point.
- **It can't live in the backend's memory.** Cloud Run scales to zero and runs several instances, so an
  in-memory denylist is wiped on restart and not shared between instances.

### Once there's a lookup, the JWT stops paying for itself

A JWT exists to avoid a lookup per request. With a denylist we do a lookup on every request anyway, but
still carry the JWT's costs:

| | JWT + denylist | Opaque token + sessions table |
| --- | --- | --- |
| Lookup per request | Yes | Yes |
| Extra service | Redis | None, Neon is our database |
| Signing secret to manage | Yes | No |
| Where expiry lives | Token `exp` and denylist TTL | One column, `expires_at` |
| Token contents | Readable, `sub` exposes the email | Meaningless random bytes |
| Library pitfalls (`alg` confusion, `none`) | Must be guarded | None |
| Token size | Hundreds of bytes | About 43 characters |

With a database row per session, the row itself is the source of truth. Expiry no longer needs to travel
inside the token, so the token can be a plain random string.

JWTs remain the right tool when many services verify tokens independently, or for 5–15 minute access
tokens paired with a revocable refresh token. Neither applies to Lyricfy today.

## Chosen design: opaque tokens in a Neon sessions table

The browser holds a random token; Neon holds only its SHA-256 hash and expiry. A session is valid while
its row exists and `expires_at` is in the future.

```mermaid
sequenceDiagram
    participant B as Browser cookie
    participant G as Go backend
    participant N as Neon sessions table

    Note over B,N: Login
    B->>G: email + password
    G->>N: INSERT sha256(token), expires_at
    G-->>B: raw token, stored in the cookie

    Note over B,N: Each request
    B->>G: Bearer token
    G->>N: SELECT by hash, expires_at > now()
    G-->>B: row: OK, no row: 401

    Note over B,N: Logout
    B->>G: POST /api/auth/logout
    G->>N: DELETE the row
    G-->>B: 200, frontend deletes the cookie
```

Only the hash and expiry ever reach Neon; the raw token lives in the cookie.

### Schema

```sql
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32), -- sha256(token)
    user_email text NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
```

The single user still lives in the backend config, so sessions record the user's email. It
becomes a `user_id` referencing a `users` table once users move into the database.

### Login

1. The browser posts the email and password.
2. The backend checks them with bcrypt.
3. The backend deletes expired sessions (see Cleanup).
4. The backend generates 32 random bytes with `crypto/rand` and base64url-encodes them. This is the token.
5. The backend inserts `sha256(token)` with `expires_at = now + 30 days` (`session.ttl`).
6. The backend returns the **raw token** and `expiresAt`.
7. The frontend stores the raw token in the `lyricfy_session` cookie, expiring at `expiresAt`.

Logging in again starts another session and leaves the others alone, so each device has its own.

### Each request

1. The frontend server sends `Authorization: Bearer <token>`.
2. The backend computes `sha256(token)` and runs
   `SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`.
3. A row means the user is logged in. No row means the token expired or was logged out, so the backend
   returns 401, and the frontend clears the cookie.

A token that isn't 32 base64url-encoded bytes is rejected without a query. If the database can't be
reached, the backend returns 500, not 401, so an outage doesn't log anyone out.

### Logout

1. The frontend calls `POST /api/auth/logout` with the token, then deletes the cookie.
2. The backend runs `DELETE FROM sessions WHERE token_hash = $1`. The token is dead from that moment, even
   if someone copied it.

The logout route needs the frontend key but not a valid session, and succeeds even for an unknown or
expired token, so logging out always works. The frontend deletes the cookie even if the backend call
fails; the session then lasts until it expires.

"Log out everywhere" isn't built yet; it would be `DELETE FROM sessions WHERE user_email = $1`.

### Cleanup

Expired rows are already invalid because of the `expires_at > now()` check, so deleting them is
housekeeping, not security. Every successful login runs `DELETE FROM sessions WHERE expires_at <= $1`
with the current time. Postgres has no native TTL, and this makes one unnecessary.

### Why hash the token, and why SHA-256

- **Hash:** if the table leaks (a backup, a log, an SQL injection), the attacker gets only hashes. A hash
  put in a cookie is hashed again by the server and matches nothing. Returning or storing the hash in the
  browser would defeat this.
- **SHA-256, not bcrypt:** bcrypt is slow on purpose because passwords are guessable. A 32-byte random
  token has 256 bits of entropy and cannot be guessed, and it is hashed on every request, so a fast hash
  is the right choice.

### Neon caveat

Neon's free tier suspends idle compute, so the first request after a quiet period can take a few hundred
milliseconds longer. The backend uses Neon's direct (unpooled) connection string with `pgx`: its
migrations take a session-level lock, which Neon's pooler doesn't support, and one Cloud Run instance
with a small pool is well within Neon's connection limit.
