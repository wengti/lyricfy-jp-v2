# Running Lyricfy JP v2 locally

## Frontend

```bash
cd frontend
npm run dev
```

### Tests

```bash
cd frontend
npm test             # run once
```

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
