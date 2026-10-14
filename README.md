# FluentPath

A roadmap-first learning platform in the spirit of roadmap.sh. Each course is a roadmap of skills. You learn each skill in a short chat lesson with an AI tutor, prove it in a rubric-scored assessment, and your progress is saved to your own dashboard. **English** (A0 → B2, with optional C1–C2) is the first course; the platform is built for more.

- **Public:** a landing page, the course catalog, and each course's roadmap and lesson details.
- **With a free account** (email + password): learning, assessments, a personal dashboard (courses, streak, 4-week activity, open lessons, scores), and progress.
- **Phones:** after login, the app uses a native-style bottom tab bar (Home · Courses · ▶ continue · Progress · Account). Desktop and logged-out visitors get the floating pill nav from [ihsan.my.id](https://ihsan.my.id).

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
packages/courses/<slug>/  One folder per course: course.json, <level>.json roadmap files, lessons/<nodeId>.json
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

Press `w` for web, `a` for Android, or `i` for iOS. Migrations run automatically. Create an account from the app's Sign up page.

### Pointing the app at the API

The app reads `EXPO_PUBLIC_API_URL` from `apps/mobile/.env`. This value is public and not secret.

| Target | URL |
|---|---|
| Web, iOS simulator | `http://localhost:8080` |
| Android emulator | `http://10.0.2.2:8080` (the default when unset) |
| Physical device | `http://<your-LAN-IP>:8080`, or an HTTPS tunnel |

When serving web from another origin, add it to `CORS_ORIGINS` in `apps/api/.env`.

### Opening it on your phone (same Wi-Fi)

The Go API can also serve the built web app, so your phone loads everything from one address. This avoids CORS and the browser's local-network restrictions.

```bash
cd apps/mobile && npx expo export -p web
```

Add `WEB_DIR=../mobile/dist` to `apps/api/.env`, restart the API, and open `http://<your-LAN-IP>:8080` on the phone. Find the IP with `ipconfig getifaddr en0`. Re-run the export after changing app code.

The API has no authentication, so anyone on the same network can use it while it's running.

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

- **Courses are fixed data.** At startup the Go API validates every course in `packages/courses` (see Courses below). The AI never changes a roadmap.
- **The tutor teaches; the server decides.** Each lesson runs through `INTRODUCTION → EXPLANATION → GUIDED_PRACTICE → INDEPENDENT_PRACTICE → ASSESSMENT`. The model may suggest `advance_phase`, but the server only honours it after a minimum number of learner replies per phase. It also advances on its own after a maximum, so a learner can't get stuck.
- **Assessment is objective-based.** The evaluator returns per-objective scores. Go validates them (every objective present, 0–100), computes the mean, and applies the threshold (80). Malformed output or a provider failure returns a retryable 502 and saves nothing.
- **Mastery happens exactly once.** Completion runs in one transaction under row locks, and repeated submissions return the existing result. A failed attempt moves the session to `REVIEW` with targeted practice, then back to assessment.
- **Locked status is derived** from prerequisites and is never stored.

## API

| Method | Endpoint | Auth | Purpose |
|---|---|---|---|
| GET | `/healthz` | – | Health and DB status |
| POST | `/api/v1/auth/register` · `/login` · `/logout` | – | Accounts. Sets an HttpOnly `fp_session` cookie (web); native clients send `X-FluentPath-Client: native` and get a bearer token |
| GET | `/api/v1/auth/me` | required | Current user |
| GET | `/api/v1/courses` | optional | Catalog; with login, enrolled courses include progress |
| GET | `/api/v1/courses/{course}/roadmap` | optional | Roadmap with your statuses (a new learner's when anonymous) |
| GET | `/api/v1/courses/{course}/nodes/{nodeId}` | optional | Node detail, objectives, prerequisites, `canStart` |
| POST | `/api/v1/courses/{course}/enroll` | required | Add the course to your dashboard (idempotent) |
| GET | `/api/v1/courses/{course}/progress` | required | Mastered skills, attempts, open lessons in that course |
| GET | `/api/v1/dashboard` | required | Courses, streak, activity, open lessons, recent scores |
| POST | `/api/v1/sessions` | required | Start or resume a lesson `{course, nodeId}` (auto-enrolls) |
| GET | `/api/v1/sessions/{id}` | required | Session state, messages, assessment items (owner only) |
| POST | `/api/v1/sessions/{id}/messages` | required | Learner reply → tutor reply `{content}` |
| POST | `/api/v1/sessions/{id}/assessment` | required | Submit `{answers:[{itemId,answer}]}` |

Errors have the shape `{"error":{"code","message","retryable"}}`. Request bodies are capped at 32 KB and messages at 2,000 characters. POSTs must be JSON, which together with SameSite=Lax cookies blocks cross-site form CSRF. Rate limits are per IP: `RATE_LIMIT_PER_MIN`, `AI_RATE_LIMIT_PER_MIN` and `AUTH_RATE_LIMIT_PER_MIN`.

### Accounts and security

- Passwords are hashed with argon2id. Session tokens are 32 random bytes; the database stores only their SHA-256. They last 30 days with a sliding expiry, and expired sessions are cleaned up hourly.
- A wrong password and an unknown email give the same error and take about the same time.
- Every lesson, attempt and progress query is scoped to the logged-in user. Another user's session ID returns 404.

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

- **Go tests** cover the course validator, status derivation, phase rules, tutor-reply parsing, and scoring with malformed evaluator output. The DB integration tests cover the full loop, a failed attempt not unlocking anything, locked and coming-soon nodes, provider failures, a concurrent double-submit mastering once, all 7 lessons end to end, user isolation, the dashboard and streak, the v1 → v2 migration, and the auth flow (register, login, logout, expiry, validation, CSRF guard). Each test runs in its own schema.
- **`e2e.sh`** builds the API and runs it against a throwaway database with the mock tutor. It checks that anonymous users can't start lessons, then signs up two users. User A masters every ready lesson over HTTP (with a failing attempt first), and user B must see none of it. It then restarts the API, logs in again, and checks that the progress persisted.

## Courses

- Each course lives in `packages/courses/<slug>/`:
  - `course.json`: title, tagline, category, tone, levels, target level, and the tutor/evaluator prompt policies.
  - `<level>.json`: the roadmap nodes for each level.
  - `lessons/<nodeId>.json`: content for every node marked `ready`.
- At startup the API validates every course: unique slugs and IDs, valid prerequisites, no cycles, a ready root, and assessment coverage for every objective. It refuses to start if a course is broken.
- English has 59 nodes across A0–C2. Seven have full lessons: alphabet & sounds, pronouns, common vocabulary, basic sentences, *to be*, simple present, and questions. The rest show as **Coming soon**.
- **Adding a course:** copy the `english` folder structure, write `course.json` and the level files, and run the Go tests. The new course then appears in the catalog automatically.

## Known limitations

- **No email verification or password reset yet.** Accounts work, but there's no "forgot password" flow and emails aren't verified.
- **Scores are not certification.** AI scores are a learning signal, not a CEFR certification, and model judgements can be wrong.
- **Voice is not implemented.** Push-to-talk and TTS (P1) are on the backlog, and there is no pronunciation scoring.
- **Android is only partly tested.** Web and the iOS simulator (Expo Go) were checked manually. Android needs an emulator/AVD, which wasn't available on the build machine.

## Next steps

1. Push-to-talk transcription (`expo-audio` → `/api/v1/audio/transcribe`), then optional tutor TTS.
2. Password reset and email verification.
3. More ready lessons (self-introduction, daily routines, everyday conversation).
4. Spaced repetition and a placement test.
5. Deploy: Go API + managed Postgres, and the Expo Web export (`npx expo export -p web`).
