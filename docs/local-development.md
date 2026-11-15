# Running Lyricfy JP v2 locally

You need Go 1.26+, Node.js 24 and Docker (for the local Postgres).

The frontend talks to the backend, so run both: the backend in one terminal, the frontend in another.
Then open http://localhost:3000 and log in with the email and password from `backend/config.local.yml`.

## Backend

The backend needs Postgres. Locally it runs in Docker (production uses Neon). Start it, and
leave it running; its data survives restarts:

```bash
cd backend
docker compose up -d
```

First time only: create `backend/config.local.yml` (gitignored) from the example and fill it in.
Its `database.url` already points at the Docker database.

```bash
cd backend
cp config.example.yml config.local.yml
openssl rand -base64 32        # paste into frontend.api_key
go run ./cmd/hashpassword      # type your password, paste the output into user.password_hash
```

The backend creates and updates its tables itself when it starts, from the numbered files in
`backend/internal/database/migrations/`. To change the schema, add a new file there; never edit
one that has been merged.

Run the API with live reload (listens on http://localhost:8080; rebuilds and restarts on every save):

```bash
cd backend
go tool air
```

Air is pinned in `go.mod`, so there's nothing to install. Without live reload: `go run ./cmd/server`.

### Tests

The database tests need the Docker database and `TEST_DATABASE_URL`; without it they're
skipped (in CI they fail instead). Each test gets its own empty schema, which is dropped
afterwards, so your development data is left alone.

```bash
cd backend
export TEST_DATABASE_URL="postgres://lyricfy:lyricfy@localhost:5432/lyricfy?sslmode=disable"
go test ./...
```

## Frontend

First time only: create `frontend/.env.local` (gitignored) from the example. `FRONTEND_API_KEY` must
be the same value as `frontend.api_key` in `backend/config.local.yml`.

```bash
cd frontend
cp .env.example .env.local
```

Run it (http://localhost:3000):

```bash
cd frontend
npm run dev
```

### Tests

```bash
cd frontend
npm test             # run once
```

## Production images (Docker)

Each app has a `Dockerfile` that builds the image deployed to Cloud Run. Neither image contains
config or secrets: the backend reads its config file from `CONFIG_PATH` (on Cloud Run, from
Parameter Manager instead) and its database from `DATABASE_URL` when set, and the frontend reads
`API_URL` and `FRONTEND_API_KEY` from the environment at runtime.

To check the images locally, run both containers on one Docker network (run from the repo root,
after the first-time setup above, with the Docker database running). Inside a container,
`localhost` is the container itself, so `DATABASE_URL` points the backend at your machine instead:

```bash
docker build -t lyricfy-api backend
docker build -t lyricfy-web frontend
docker network create lyricfy

# MSYS_NO_PATHCONV=1 stops Git Bash on Windows rewriting /config/... into a Windows path
# (harmless elsewhere)
MSYS_NO_PATHCONV=1 docker run --rm --name api --network lyricfy \
  -v "$(pwd)/backend/config.local.yml:/config/config.yml:ro" -e CONFIG_PATH=/config/config.yml \
  --add-host host.docker.internal:host-gateway \
  -e DATABASE_URL="postgres://lyricfy:lyricfy@host.docker.internal:5432/lyricfy?sslmode=disable" \
  lyricfy-api

# In another terminal; FRONTEND_API_KEY must equal frontend.api_key in config.local.yml
docker run --rm --network lyricfy -p 3000:3000 \
  -e API_URL=http://api:8080 -e FRONTEND_API_KEY=... \
  lyricfy-web
```

Then open http://localhost:3000.
