# Lyricfy JP v2

**Learn Japanese through music you actually love.** Now rebuilt on a cleaner, tested and continuously deployed foundation.

Lyricfy JP connects to Spotify and shows live, karaoke-style lyrics with furigana and English translations. You can save words from the lyrics to a personal dictionary and practise them with flashcards.

v2 is a full rewrite of [Lyricfy JP v1](https://github.com/wengti/lyricfy-jp). It's a personal app, built for a single user (me).

> **Status:** planning and initial scaffold. Nothing is deployed yet. See [Roadmap](#roadmap).

---

## Why a rewrite?

v1 works, but it grew feature by feature without a safety net:

- **Bugs.** Many fixes were patches on top of patches (see v1's git history), and some behaviour still isn't right.
- **No CI/CD.** No automated tests and no pipeline. Deploys depended on Vercel's git integration, and the only check was a local `tsc` hook before commits.
- **Database changes by hand.** The schema is a single `schema.sql` pasted into the Supabase SQL editor, with no migration history.
- **Mixed data formats.** For example, `lyrics_cache.translated_lines` holds either a legacy plain array or a newer object, and the code branches on which one it finds.
- **Built for many users, used by one.** v1 has per-user encrypted API keys, per-user Spotify apps, admin roles and RLS policies. That's extra complexity a single-user app doesn't need.

### Goals for v2

1. **Migrate v1's features one at a time**, fixing their bugs along the way rather than copying them over.
2. **A proper CI/CD pipeline:** tests, Docker images and automated deploys on every merge.
3. **Infrastructure as code:** every cloud resource defined in Terraform.
4. **A clean structure** with a clear frontend/backend split, so new features (such as an AI quiz) are easy to add.

---

## Features

The v1 column shows what exists in v1 and is due to be migrated.

| Area | Feature | v1 | Notes for v2 |
|---|---|:-:|---|
| **Auth** | Email + password login | ✅ (Supabase Auth) | Single user only. Credentials stored hashed in our own Postgres, login handled by the Go backend |
| **Lyrics** | Find lyrics for the current Spotify song | ✅ | Compare the current method (lrclib.net → Genius scraping → manual paste) with alternatives, and look for a better search for Japanese songs |
| | Reject non-Japanese songs | ⚠️ partial | **Always** reject songs without Japanese characters |
| | Always re-check the source | ❌ | Always query the lyrics API to see whether a song can now be found, rather than trusting a stale cache entry |
| | Furigana + English translation | ✅ | LLM call, one request for both |
| | Sync with Spotify playback | ✅ | Karaoke-style highlighting, click a line to seek |
| | Spotify player controls | ✅ | Play/pause, seek, next/previous (needs Spotify Premium) |
| | Highlight lyrics to save to the dictionary | ✅ | AI fills in reading, translation and an example sentence; can split a phrase into words |
| **Lyrics library** | Save lyrics to the library | ✅ (as a shared cache) | Becomes an explicit library |
| | Search | ✅ | |
| | Sort | ❌ | New |
| **Dictionary** | Add a new entry | ✅ | Manual add with AI enrichment |
| | Search | ✅ | |
| | Browse | ✅ | Sort by success rate or date, filter by tag |
| **Quiz** | Flashcards | ✅ | JP → EN and EN → JP; study sets (newest, random, weakest, custom); 3D flip; session summary |
| | AI quiz | ❌ | To be designed |

---

## Architecture

```
            Browser
               │  (login: email + password)
               ▼
┌──────────────────────────────┐
│  web/   Next.js (standalone) │  Cloud Run, public
└──────────────┬───────────────┘
               │  Google-signed identity token (service-to-service auth)
               ▼
┌──────────────────────────────┐        ┌───────────────────────┐
│  api/   Go + Echo            │───────►│  Neon (Postgres)      │
│                              │        └───────────────────────┘
│                              │───────►  OpenRouter → Claude Sonnet 5.5
│                              │───────►  Spotify Web API, lyrics sources
└──────────────────────────────┘
   Cloud Run, private (only the web service can call it)

Config + secrets: GCP Parameter Manager (references to Secret Manager)
Infrastructure:   Terraform (GCP + Neon), state in a GCS bucket
CI/CD:            GitHub Actions → Artifact Registry → Cloud Run
```

| Layer | Choice |
|---|---|
| Frontend | Next.js (`output: 'standalone'`) in Docker on **Cloud Run**, public |
| Backend | **Go + Echo** in Docker on **Cloud Run**, private: only the frontend's service account can call it |
| Database | **Neon** serverless Postgres. |
| LLM | **OpenRouter → Claude Sonnet 5.5**, with a credit limit on the key |
| Config & secrets | **GCP Parameter Manager**: a versioned YAML config, edited in the console, whose secret values are references to Secret Manager. Loaded once at startup |
| Infrastructure | **Terraform** (Google + Neon providers). State in a hardened GCS bucket created by a one-time bootstrap config |
| CI/CD | **GitHub Actions**: one pipeline that tests, builds images tagged with the commit SHA, and deploys through `terraform plan` → manual approval → `terraform apply`. Workload Identity Federation, so no stored cloud keys |
| Cost guardrails | GCP budget alert ($1) and OpenRouter credit limit |

### Design decisions

- **Hosting is free; only the LLM costs money.** Cloud Run, Neon, GCS and GitHub Actions all stay within free tiers for one user. Both services scale to zero, so expect a short delay on the first request after a while idle.
- **The backend is never public.** The frontend calls it with a Google-signed identity token, so nobody else can reach the API or spend LLM credits.
- **Every change ships the same way.** CI builds and pushes the images, then Terraform deploys them along with any infrastructure change, after a human approves the plan. Nothing is deployed by hand.
- **Secrets stay out of images and out of Git.** Terraform creates the secret containers, and the values are added with `gcloud`. The Neon connection string does pass through Terraform state, which is why the state bucket is locked down.
- **Single user, by design.** No per-user API keys, no per-user Spotify apps, no admin roles. Spotify and OpenRouter credentials are app config.


---

## Infrastructure

Every cloud resource is defined in Terraform under [`infrastructure/`](infrastructure/). You apply
`bootstrap/` once from your machine; the pipeline applies everything else.

| Path | Applied by | What it manages |
|---|---|---|
| `bootstrap/` | you, once (local state, gitignored) | Only what the pipeline needs before its first run: the Terraform state bucket, the pipeline's identities (`ci.tf`), the APIs, the image registry, and the secret containers and config parameter that you fill in by hand |
| `infrastructure/` (root) | the pipeline (state in the bucket) | The services' identities and **every permission** they get (`identities.tf`), the two modules below, and the budget alert (`budget.tf`) |
| `backend/` | (module of the root) | The backend's Cloud Run service (private) |
| `frontend/` | (module of the root) | The frontend's Cloud Run service (public), and its permission to call the backend |

The rule: something goes in `bootstrap/` if the pipeline needs it in order to run, or if you have
to fill it in by hand before the first deploy. Everything else goes through the pipeline,
including new resources and the permissions to use them.

**Backend config** lives in Parameter Manager as the `backend-config` parameter: the same YAML as
`backend/config.local.yml`, edited in the console. Secret values (`frontend.api_key`,
`session.secret`, `user.password_hash`) aren't stored in it: it refers to Secret Manager secrets
with `__REF__(...)`, and Parameter Manager fills them in when the backend loads the newest version
at startup. Locally, the backend still reads `config.local.yml`.

### First-time setup

#### 0. Prerequisites

- [gcloud CLI](https://cloud.google.com/sdk/docs/install) and [Terraform](https://developer.hashicorp.com/terraform/install) 1.10+
- A GCP project with a billing account linked to it. Billing is required even to use the free tier.
- Billing Account Administrator on that billing account (you have it if you created it): bootstrap
  lets the pipeline manage the budget alert there

Log in. Terraform uses the application-default credentials, and the quota project lets APIs such
as Cloud Billing accept them:

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project <project-id>
gcloud auth application-default set-quota-project <project-id>
```

#### 1. Apply bootstrap

```bash
cd infrastructure/bootstrap
cp terraform.tfvars.example terraform.tfvars   # fill in project_id and billing_account
terraform init
terraform apply
```

Its state stays in `bootstrap/terraform.tfstate`. Keep it: if it's lost, every resource has to be
brought back with `terraform import` before bootstrap can be changed again.

#### 2. Add the secrets and the backend config

The three secret values (`printf`, not `echo`: a trailing newline would become part of the value):

```bash
printf '%s' "$(openssl rand -base64 32)" | gcloud secrets versions add frontend-api-key --data-file=-
printf '%s' "$(openssl rand -base64 32)" | gcloud secrets versions add session-secret --data-file=-
cd backend && go run ./cmd/hashpassword | tr -d '\n' | gcloud secrets versions add user-password-hash --data-file=- && cd ..
```

Then the config: in the console, open Secret Manager → **Parameter Manager** → `backend-config` →
**Versions** → **New version**, and paste
[`backend/config.parameter.example.yml`](backend/config.parameter.example.yml) with `PROJECT_ID`
and your email filled in. Its secret values are references (`__REF__(...)`) to the secrets above,
filled in when the backend reads it.

#### 3. Connect GitHub

In the repository's Settings → Secrets and variables → Actions → **Variables**, add:

| Variable | Value |
|---|---|
| `GCP_PROJECT_ID` | your project ID |
| `GCP_REGION` | `terraform -chdir=infrastructure/bootstrap output -raw region` |
| `GCP_BILLING_ACCOUNT` | your billing account ID |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | `terraform -chdir=infrastructure/bootstrap output -raw workload_identity_provider` |

They aren't secrets: Google only accepts tokens from this repository's `main` branch. The
pipeline works out its service account emails from the project ID.

Then in Settings → **Environments**, create an environment named `production` and add yourself
under **Required reviewers**. This is the approval step before every apply. Leave "Prevent
self-review" off, or you can't approve your own merges. (Required reviewers are free on public
repositories; private ones need GitHub Pro or above.)

#### 4. First deploy

In the Actions tab, run the **Pipeline** workflow on `main` (or merge anything to `main`). Approve
the plan when asked; the apply job's link is the public site.

### Deployment pipeline

[`.github/workflows/pipeline.yml`](.github/workflows/pipeline.yml) runs on every pull request and
every push to `main`, except changes that only touch Markdown or `docs/`:

```
test (backend ┃ frontend ┃ infrastructure) → build image (backend ┃ frontend) → plan → approve → apply
```

| Stage | What it does | Pull request | `main` |
|---|---|:-:|:-:|
| **Test** | In parallel; one failure cancels the rest. Backend: `gofmt`, `go vet`, `go test -race`. Frontend: ESLint, Vitest. Infrastructure: `terraform fmt` and `validate` | ✅ | ✅ |
| **Build image** | Both Docker images in parallel; one failure cancels the other. On `main` they're pushed, tagged with the commit SHA | build only | build + push |
| **Plan** | `terraform plan` with the new image tags, saved and shown on the run's summary page | | ✅ |
| **Approve** | The run pauses until a `production` reviewer approves (skipped if the plan has no changes) | | ✅ |
| **Apply** | Applies exactly the approved plan: new images and any infrastructure change together | | ✅ |

To review, open the waiting run: the plan is on its summary page. Approve with **Review
deployments**, or reject to stop. Terraform refuses to apply a plan if anything changed since it
was made, and only one run per branch goes at a time, so an approved plan is always current.

Only `main` can reach the cloud: GitHub signs in with Workload Identity Federation (no stored
keys), and Google only trusts this repository's `main` branch. Each stage signs in as its own
service account, with only what that stage needs:

| Stage | Service account | Can | Usable by |
|---|---|---|---|
| Build | `github-build` | Push images | Any job on `main` |
| Plan | `github-plan` | Read everything, change nothing | Any job on `main` |
| Apply | `github-apply` | Anything (Owner) | Only jobs in the `production` environment, so only after approval |

Google enforces the last column, so code that reaches `main` can push an image or read the
project, but can't change anything until someone approves the plan. Since the apply account is
Owner, **the approval is the safeguard: read the plan before approving.**

Each merge builds and deploys both images, so both services get a new revision even if only one
changed.

### Day to day

- **Ship a change (code or infrastructure):** merge to `main`, then approve the plan. To roll
  back, revert the commit and do the same.
- **Change the backend config:** in Parameter Manager, open `backend-config` → **Versions** →
  **New version**, and paste the edited YAML (start from the current version). Then redeploy so
  the backend loads it: Cloud Run → `lyricfy-backend` → **Edit & deploy new revision** →
  **Deploy**. No pipeline run or approval needed. Then delete old versions to stay within the
  free tier (6 versions); keep the previous one if you might roll back. To roll back, create a new
  version with the old content: don't disable the newest version, because the backend always
  loads the newest one and fails to start if it's disabled.
- **Rotate a secret:** add a new version of it in Secret Manager (as in setup step 2), redeploy the
  backend as above (or, for `frontend-api-key`, the frontend too), then destroy the old version:
  `gcloud secrets versions destroy <n> --secret=<name>`. Old versions count toward the free tier.
- **Add a resource, or give a service access to one:** add the resource to `infrastructure/` and
  the permission to `infrastructure/identities.tf` (enabling its API with `google_project_service`
  if it's a new kind of Google service), then merge and approve. No local step.
- **Change bootstrap:** only needed to change the pipeline's own identities, or to add another
  secret or parameter that you fill in by hand. Edit it and `terraform apply` locally, like the
  first time.
- **Plan locally:** in `infrastructure/`, copy `terraform.tfvars.example` to `terraform.tfvars`,
  `terraform init -backend-config="bucket=<project-id>-tfstate"`, then `terraform plan`. Leave
  applying to the pipeline.
- **Tear down:** `terraform destroy` in `infrastructure/` (with the current image tags as
  variables), then in `bootstrap/`. The state bucket has `prevent_destroy`, so remove that first
  if you really mean it.

### Staying free

Everything here fits in GCP's free tier for a single user:

| Service | What keeps it free |
|---|---|
| Cloud Run | Scales to zero, CPU billed only during requests (`cpu_idle`), at most 1 instance per service |
| Artifact Registry | Cleanup policy keeps the 2 newest versions of each image (free tier: 0.5 GB) |
| Secret Manager | 3 secrets, one version each (free tier: 6 active versions, 10,000 accesses a month) |
| Parameter Manager | 1 parameter; delete old versions after edits (free tier: 6 versions, 10,000 reads a month). The backend reads it once per cold start |
| Cloud Storage | Only Terraform state, in a free-tier region (us-central1/us-east1/us-west1) |
| GitHub Actions | Free for public repositories; private ones get 2,000 minutes a month. Docker layers are cached between runs |
| Budget alert | Free; emails the billing account admins at 10%, 50% and 100% of the monthly budget, and when spend is forecast to pass 100% |

The budget alert only alerts: it never stops anything, and billing data lags by several hours.
Its amount is in the billing account's currency (`budget_amount` in `infrastructure/variables.tf`).

**If a budget alert arrives**, take the site offline by blocking all outside traffic to both
services. Cloud Run only bills while handling requests, so this stops its costs within seconds,
and nothing is deleted:

```bash
gcloud run services update lyricfy-frontend --region asia-southeast1 --ingress internal
gcloud run services update lyricfy-backend  --region asia-southeast1 --ingress internal
```

Then find the cause in the console under Billing → Reports, grouped by SKU. To bring the site
back, run the same commands with `--ingress all`. Until the cause is fixed, don't merge to `main`
(or reject the approval step): the next apply restores normal traffic, because Terraform puts back
what the code says.

Regions:

- **`asia-southeast1` (Singapore)** for the registry, secrets and both services: the closest
  region to Malaysia, which has no GCP region yet. Set by `region` in `bootstrap/` and
  `GCP_REGION` in GitHub, which must match. Cloud Run, Artifact Registry and Secret Manager have
  no region limit on their free tiers, apart from Cloud Run's free 1 GB of outbound data, which
  only covers traffic from North America. So the pages sent to the browser are billed, but for
  one user that's about a cent a month. The registry and services share the region, so images are
  never pulled across regions, which would be billed.
- **`us-central1`** for the state bucket only (`state_bucket_location` in `bootstrap/`). Cloud
  Storage's free tier covers only `us-central1`, `us-east1` and `us-west1`, and the bucket is only
  touched when Terraform runs, so its distance doesn't matter.
