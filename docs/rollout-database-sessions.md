# Rollout: Neon database and database sessions

A one-time checklist for deploying the change that adds the Neon database and moves sessions into
it (see [session-tokens.md](session-tokens.md)). Work through it top to bottom, then delete this
file. A fresh setup doesn't need it: the root README's first-time setup already covers Neon.

Commands run from the repo root in Git Bash unless they say otherwise.

## Before merging

- [ ] **1. Create a Neon organization API key.** Sign up at [neon.com](https://neon.com) (the Free
  plan is enough). In the Neon Console, open your organization's **Settings** → **API keys** and
  create an **organization** key, not a personal one: Terraform creates the project in the key's
  organization. Keep it open for step 3; you can't view it again.

- [ ] **2. Apply bootstrap locally.** It creates the `neon-api-key` secret and lets the pipeline's
  plan stage read it. First make sure gcloud points at this project (see "First-time setup" in
  the README if you've used another project since):

  ```bash
  gcloud config set project <project-id>
  gcloud auth application-default set-quota-project <project-id>
  cd infrastructure/bootstrap
  terraform init
  terraform apply
  ```

  The plan should add exactly 2 resources and change nothing else:
  `google_secret_manager_secret.this["neon_api_key"]` and
  `google_secret_manager_secret_iam_member.plan_neon_api_key`.

- [ ] **3. Store the key.** Paste it at the hidden prompt (it stays out of your shell history),
  then check there's one enabled version:

  ```bash
  read -rsp "Neon API key: " key && printf '%s' "$key" | gcloud secrets versions add neon-api-key --data-file=-; unset key
  gcloud secrets versions list neon-api-key
  ```

- [ ] **4. Update your local setup.** The backend now needs Postgres:

  ```bash
  cd backend
  docker compose up -d
  ```

  Add the `database` section from `config.example.yml` to your `config.local.yml`, and delete its
  `session.secret` line (no longer used). Then check the backend starts with `go tool air`: the
  log should say `"database ready"`.

- [ ] **5. Open the pull request** and let its checks pass. The backend test job now starts its
  own Postgres for the database tests.

## Deploying

- [ ] **6. Merge, then review the plan before approving.** On the run's summary page, expect:

  | Change | Resources |
  |---|---|
  | Create | `module.database.neon_project.this`, `module.database.google_secret_manager_secret.database_url`, `module.database.google_secret_manager_secret_version.database_url`, `google_secret_manager_secret_iam_member.backend_database_url`, `google_secret_manager_secret_iam_member.plan_database_url` |
  | Replace | `time_sleep.iam_propagation` (it gained a trigger; it just waits 60 seconds again) |
  | Update | both Cloud Run services: new images, and the backend gains the `DATABASE_URL` env var |

  Nothing should be destroyed apart from the `time_sleep` replacement. If the **plan** fails
  reading `neon-api-key`, steps 2–3 are missing: do them, then re-run the workflow from the
  Actions tab.

- [ ] **7. Check it worked.**
  - Cloud Run → `lyricfy-backend` → **Logs**: look for `"database ready"` with
    `"migrations_applied":1`.
  - Open the site. You'll be sent to the login page once, because your old JWT cookie is no
    longer accepted. Log in.
  - Neon Console → your project → **SQL Editor**:

    ```sql
    select * from goose_db_version;              -- versions 0 and 1
    select user_email, expires_at from sessions; -- one row, expiring in 30 days
    ```

  - Log out, run the `sessions` query again (your row is gone), then log back in.

## After deploying: retire `session.secret`

The new backend ignores `session.secret` but still accepts it, so the live config keeps working.
Remove it in this order; doing it earlier would stop the old backend from starting.

- [ ] **8. Remove it from `backend-config`.** Secret Manager → **Parameter Manager** →
  `backend-config` → **Versions** → **New version**: start from the current version and delete
  the `secret: '__REF__(...session-secret...)'` line under `session:` (keep `ttl`). Then
  redeploy the backend (Cloud Run → `lyricfy-backend` → **Edit & deploy new revision** →
  **Deploy**) and check its logs say `"database ready"`. Delete the old parameter versions, as the
  README's "Day to day" section describes.

- [ ] **9. Remove the leftovers in code**, in a follow-up pull request (Claude can make it):
  - `session_secret` in the `local.secrets` of `infrastructure/main.tf` (the backend config's
    permission to read it)
  - the deprecated `Secret` field in `backend/internal/config/config.go`, and its test
  - `session_secret` in `infrastructure/bootstrap/secrets.tf`
  - the notes about the transition: the `session-secret` line under "2. Add the secrets and the
    backend config" in the root README, and the "An older `config.local.yml`" paragraph in
    `docs/local-development.md`

  Merge and approve that first, so the pipeline removes the permission. **Then** run
  `terraform apply` in `infrastructure/bootstrap`, which deletes the `session-secret` secret.
  The other way round, the pipeline's state would still hold a permission on a secret that no
  longer exists.

- [ ] **10. Delete this file.**
