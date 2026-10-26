# Running Lyricfy JP v2 locally

The frontend talks to the backend, so run both: the backend in one terminal, the frontend in another.
Then open http://localhost:3000 and log in with the email and password from `backend/config.local.yml`.

## Backend

First time only: create `backend/config.local.yml` (gitignored) from the example and fill it in.

```bash
cd backend
cp config.example.yml config.local.yml
openssl rand -base64 32        # run twice: one for frontend.api_key, one for session.secret
go run ./cmd/hashpassword      # type your password, paste the output into user.password_hash
```

Run the API with live reload (listens on http://localhost:8080; rebuilds and restarts on every save):

```bash
cd backend
go tool air
```

Air is pinned in `go.mod`, so there's nothing to install. Without live reload: `go run ./cmd/server`.

### Tests

```bash
cd backend
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
