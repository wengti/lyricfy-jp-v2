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
| Config & secrets | **GCP Parameter Manager**: a versioned JSON config whose secret values (Neon URL, OpenRouter key, Spotify credentials) are references to Secret Manager. Loaded once at startup |
| Infrastructure | **Terraform** (Google + Neon providers). State in a hardened GCS bucket created by a one-time bootstrap config |
| CI/CD | **GitHub Actions** with Workload Identity Federation (no stored cloud keys), images tagged with the commit SHA, workflows filtered by path |
| Cost guardrails | GCP budget alert ($1) and OpenRouter credit limit |

### Design decisions

- **Hosting is free; only the LLM costs money.** Cloud Run, Neon, GCS and GitHub Actions all stay within free tiers for one user. Both services scale to zero, so expect a short delay on the first request after a while idle.
- **The backend is never public.** The frontend calls it with a Google-signed identity token, so nobody else can reach the API or spend LLM credits.
- **Terraform owns the infrastructure; CI owns the code version.** Terraform creates the Cloud Run services and ignores image changes (`lifecycle.ignore_changes`), and CI deploys new images.
- **Secrets stay out of images and out of Git.** Terraform creates the secret containers, and the values are added with `gcloud`. The Neon connection string does pass through Terraform state, which is why the state bucket is locked down.
- **Single user, by design.** No per-user API keys, no per-user Spotify apps, no admin roles. Spotify and OpenRouter credentials are app config.

