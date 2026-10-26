# FluentPath

A roadmap-first English course for Indonesian adults (A0 → C2), in English and Indonesian UI: hand-written step-by-step
lessons, an AI tutor (hints, Ask AI, similar questions), and AI-written checkpoints that decide mastery. See `README.md`.

- `apps/api/` — Go API (chi, pgx), PostgreSQL; serves the web build from `apps/mobile/dist` on :8080.
- `apps/mobile/` — Expo app (Android · iOS · Web). Expo rules: `apps/mobile/AGENTS.md`.
- `packages/courses/<slug>/` — course content: roadmap levels, `lessons/`, `plans/` (hand-written steps), `audio/`.

## Writing or editing lesson content

**Follow `packages/courses/CONTENT_GUIDE.md`.** In short: write for a super-beginner reading Indonesian; never show an example that
seems to break the rule without saying why (*a university*, *an hour*); explain jargon in plain Indonesian; name the
Indonesian-vs-English traps; every question has exactly one right answer; both languages have identical questions; validate with
`go run ./cmd/coursecheck`.

## Rules

- **API keys stay server-side.** The owner fills `apps/api/.env` themselves: never print, paste or ask for a key, and never put one
  in the Expo app or an `EXPO_PUBLIC_*` variable.
- **Lesson audio is made only after the owner has reviewed the content.** New lessons go into `audio/pending.json`; run
  `bundle-audio` only once they're approved. The server never calls text-to-speech for lesson sentences by itself.
- **Wording-only edits to a lesson keep learners' progress**; changing steps or questions restarts the lesson for learners on
  the old version. Prefer wording fixes.
- Commit only when asked; keep commits focused.

## Commands

```bash
# Postgres (Docker/Colima, port 5433)
docker compose -f infra/docker-compose.yml up -d db

# API tests (DB tests need the test URL)
cd apps/api && TEST_DATABASE_URL='postgres://fluentpath:fluentpath@localhost:5433/fluentpath?sslmode=disable' go test ./...

# Course validation (as the API does at startup)
cd apps/api && go run ./cmd/coursecheck

# End to end: every ready lesson mastered over HTTP with the mock AI
scripts/e2e.sh

# App checks and the web build the API serves
cd apps/mobile && npx tsc --noEmit && npx expo lint && npx expo export -p web

# Run the API (reads apps/api/.env)
cd apps/api && go build -o bin/server ./cmd/server && ./bin/server

# Record reviewed lesson audio
cd apps/api && go run ./cmd/server bundle-audio
```
