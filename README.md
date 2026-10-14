# FluentPath

A roadmap-first English learning app with an AI tutor. Learners see every skill from A0 to C2. Each skill is taught in a short interactive chat lesson, proven in a rubric-scored assessment, and saved as durable progress that unlocks the next skills.

The core loop is **roadmap → lesson → assessment → saved mastery → next node unlocked**.

| Layer | Tech |
|---|---|
| App (Android · iOS · Web) | Expo SDK 57, React Native 0.86, Expo Router, TypeScript, TanStack Query, Zustand, react-native-svg |
| API | Go 1.27, chi, pgx v5 |
| Database | PostgreSQL 16 (Docker Compose) |
| AI | OpenAI Responses API with strict JSON-schema output, plus a deterministic `mock` provider |

The visual design follows [ihsan.my.id](https://ihsan.my.id) (`../portofolio-web`): a warm dotted canvas, white bento cards, Geist + Geist Mono, a violet accent with a violet → magenta → orange gradient, pastel tiles, a floating pill nav and an ink CTA card. The tokens live in `apps/mobile/src/theme/tokens.ts`.

## Repository layout

```
apps/mobile/            Expo app (routes in src/app, UI kit in src/components/ui)
apps/api/               Go API (cmd/server, internal/*, migrations/)
packages/curriculum/    Versioned curriculum: a0…c2.json skeleton + lessons/<nodeId>.json
infra/docker-compose.yml  PostgreSQL (+ optional api service)
scripts/e2e.sh          HTTP end-to-end check of the learning loop
```

## Quick start

Prerequisites: Node 20+, Go 1.27, and Docker (or Colima).

```bash
docker compose -f infra/docker-compose.yml up -d db
```

```bash
cp .env.example apps/api/.env      # AI_PROVIDER=mock works without any key
```

```bash
cd apps/api && go run ./cmd/server
```

In a second terminal:

```bash
cd apps/mobile && npm install && npx expo start
```

Press `w` for web, `a` for Android, or `i` for iOS. Migrations run automatically, and a single demo learner is seeded.

### Pointing the app at the API

The app reads `EXPO_PUBLIC_API_URL` from `apps/mobile/.env`. This value is public and not secret.

| Target | URL |
|---|---|
| Web, iOS simulator | `http://localhost:8080` |
| Android emulator | `http://10.0.2.2:8080` (the default when unset) |
| Physical device | `http://<your-LAN-IP>:8080`, or an HTTPS tunnel |

When serving web from another origin, add it to `CORS_ORIGINS` in `apps/api/.env`.

### Using the real AI tutor

Set these in `apps/api/.env`. Never put keys in the Expo app or in `EXPO_PUBLIC_*` variables.

```
AI_PROVIDER=openai
OPENAI_API_KEY=sk-…
OPENAI_MODEL=gpt-6-luna          # tutor replies
OPENAI_EVAL_MODEL=gpt-6.1-sol    # assessment scoring
```

Check model IDs and prices at https://developers.openai.com/api/docs/models before use.

## How progression works

- **The curriculum is fixed data.** At startup, the Go API validates `packages/curriculum`: unique IDs, valid prerequisites, no cycles, a ready root, and every objective of a ready node covered by an assessment item. The API refuses to start if any check fails.
- **The tutor teaches; the server decides.** Each lesson runs through `INTRODUCTION → EXPLANATION → GUIDED_PRACTICE → INDEPENDENT_PRACTICE → ASSESSMENT`. The model may suggest `advance_phase`, but the server only honours it after a minimum number of learner replies per phase. It also advances on its own after a maximum, so a learner can't get stuck.
- **Assessment is objective-based.** The evaluator returns per-objective scores. Go validates them (every objective present, 0–100), computes the mean, and applies the threshold (80). Malformed output or a provider failure returns a retryable 502 and saves nothing.
- **Mastery happens exactly once.** Completion runs in one transaction under row locks, and repeated submissions return the existing result. A failed attempt moves the session to `REVIEW` with targeted practice, then back to assessment.
- **Locked status is derived** from prerequisites and is never stored.

## API

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/healthz` | Health and DB status |
| GET | `/api/v1/curriculum` | Nodes and prerequisites |
| GET | `/api/v1/roadmap` | Curriculum with learner statuses and summary |
| GET | `/api/v1/nodes/{nodeId}` | Detail, objectives, prerequisites/successors, `canStart` |
| POST | `/api/v1/sessions` | Start or resume a lesson `{nodeId}` |
| GET | `/api/v1/sessions/{id}` | Session state, messages, assessment items |
| POST | `/api/v1/sessions/{id}/messages` | Learner reply → tutor reply `{content}` |
| POST | `/api/v1/sessions/{id}/assessment` | Submit `{answers:[{itemId,answer}]}` |
| GET | `/api/v1/progress` | Mastered skills, attempts, open sessions, weekly summary |

Errors have the shape `{"error":{"code","message","retryable"}}`. Request bodies are capped at 32 KB and messages at 2,000 characters. Rate limits are per IP (`RATE_LIMIT_PER_MIN`, `AI_RATE_LIMIT_PER_MIN`).

## Tests

```bash
cd apps/api && TEST_DATABASE_URL='postgres://fluentpath:fluentpath@localhost:5433/fluentpath?sslmode=disable' go test -race ./...
```

```bash
scripts/e2e.sh
```

```bash
cd apps/mobile && npx tsc --noEmit && npx expo lint
```

- **Go tests** cover the curriculum validator, status derivation, phase rules, tutor-reply parsing, and scoring with malformed evaluator output. The DB integration tests cover the full loop, a failed attempt not unlocking anything, locked and coming-soon nodes, provider failures, a concurrent double-submit mastering once, and all 7 lessons end to end. Each test runs in its own schema.
- **`e2e.sh`** builds the API and runs it against a throwaway database with the mock tutor. It masters every ready lesson over HTTP (with a failing attempt first), restarts the API, and checks that the progress persisted.

## Curriculum

- 59 nodes across A0–C2, of which 7 have full lesson content: alphabet & sounds, pronouns, common vocabulary, basic sentences, *to be*, simple present, and questions. The rest are labelled **Coming soon** and can't be started.
- To add a lesson:
  1. Write `packages/curriculum/lessons/<nodeId>.json` (intro, key points with Indonesian glosses, guided/independent practice, and assessment items for every objective).
  2. Set the node's `contentStatus` to `ready`.
  3. Run the Go tests. The validator will tell you what's missing.

## Known limitations

- **Single demo learner, no authentication.** Keep the API private. Add real auth and per-user authorization before exposing write endpoints.
- **Scores are not certification.** AI scores are a learning signal, not a CEFR certification, and model judgements can be wrong.
- **Voice is not implemented.** Push-to-talk and TTS (P1) are on the backlog, and there is no pronunciation scoring.
- **Android is only partly tested.** Web and the iOS simulator (Expo Go) were checked manually. Android needs an emulator/AVD, which wasn't available on the build machine.

## Next steps

1. Push-to-talk transcription (`expo-audio` → `/api/v1/audio/transcribe`), then optional tutor TTS.
2. Authentication and per-user data.
3. More ready lessons (self-introduction, daily routines, everyday conversation).
4. Spaced repetition and a placement test.
5. Deploy: Go API + managed Postgres, and the Expo Web export (`npx expo export -p web`).
