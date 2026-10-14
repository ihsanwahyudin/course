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

To run the API in Docker too: `docker compose -f infra/docker-compose.yml --profile api up -d --build`. The database (`pgdata`) and the lesson audio cache (`tts`, mounted at `/app/data/tts`) live in named volumes, so restarts and rebuilds keep them. Only `docker compose down -v` deletes them.

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
- **The tutor guides; the server decides.**
  - A lesson is a thread of steps shown one at a time (see *Step-by-step lessons* below). The AI writes the steps for each learner from the lesson's curated material, and Go validates them.
  - The checkpoint opens once every step is done, and passing it moves the lesson to `MASTERED`.
  - The tutor never decides mastery.
- **Assessment is objective-based.** The evaluator returns per-objective scores. Go validates them (every objective present, 0–100), computes the mean, and applies the threshold (80). Malformed output or a provider failure returns a retryable 502 and saves nothing.
- **Answers that exactly match a reference answer are scored locally.** Only the remaining items go to the AI evaluator, and if every answer matches, the AI isn't called at all.
- **Mastery happens exactly once.** Completion runs in one transaction under row locks, and repeated submissions return the existing result. A failed attempt keeps the lesson open and shows the weak objectives.
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
| GET · PUT | `/api/v1/courses/{course}/profile` | required | Onboarding questions and your answers · save `{answers, targetLevel?}` or `{skip: true}` (enrolls) |
| GET | `/api/v1/courses/{course}/progress` | required | Mastered skills, attempts, open lessons in that course |
| GET | `/api/v1/dashboard` | required | Courses, streak, activity, open lessons, recent scores |
| POST | `/api/v1/sessions` | required | Start or resume a lesson `{course, nodeId}` (auto-enrolls) |
| GET | `/api/v1/sessions/{id}` | required | Session state: steps so far, checkpoint items, summary (owner only) |
| POST | `/api/v1/sessions/{id}/steps/{n}/next` | required | Move past step `n` (a question must be answered or given up first) |
| POST | `/api/v1/sessions/{id}/steps/{n}/answer` | required | Answer `{choice}` or `{tokens}` (`similar: true` for the similar question) |
| POST | `/api/v1/sessions/{id}/steps/{n}/hint` · `/reveal` | required | A hint (max 2) · give up after both hints (`{similar: true}` optional) |
| POST | `/api/v1/sessions/{id}/steps/{n}/similar` | required | One similar question, only after a wrong answer or giving up |
| POST | `/api/v1/sessions/{id}/steps/{n}/ask` | required | Ask the tutor about this step `{content}` |
| POST | `/api/v1/sessions/{id}/assessment` | required | Submit the checkpoint `{answers:[{itemId,answer}]}` |
| GET | `/api/v1/tts?text=` | required | MP3 of an English sentence (≤300 chars, cached); 501 without a voice provider |
| POST | `/api/v1/transcribe` | required | Raw `audio/*` body (≤2 MB) → `{text}` |
| GET · PATCH · DELETE | `/api/v1/me/memory` | required | What the tutor remembers · `{enabled}` · delete all |
| PATCH · DELETE | `/api/v1/me/memory/{id}` | required | Edit `{text}` · delete one note |
| GET · POST · DELETE | `/api/v1/notes` · `/api/v1/notes/{id}` | required | Saved cheat sheets ("My notes") |
| POST | `/api/v1/flashcards` | required | Save cards `{course, nodeId, cards:[{front,back}]}` (max 20 per call, 1,000 per learner) |
| GET | `/api/v1/flashcards/due?course=` | required | Cards due for review |
| POST · DELETE | `/api/v1/flashcards/{id}/review` · `/api/v1/flashcards/{id}` | required | Grade a card `{grade: again\|good\|easy}` · delete it |

Errors have the shape `{"error":{"code","message","retryable"}}`. Request bodies are capped at 32 KB and messages at 2,000 characters. POST/PUT/PATCH bodies must be JSON, which together with SameSite=Lax cookies blocks cross-site form CSRF. Rate limits are per IP: `RATE_LIMIT_PER_MIN`, `AI_RATE_LIMIT_PER_MIN` and `AUTH_RATE_LIMIT_PER_MIN`.

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

- **Go tests** cover the course validator, status derivation, the lesson-plan validator (and the fallback plan for every lesson), leaking hints, tutor-reply parsing, and scoring with malformed evaluator output. The DB integration tests cover the step rules (answers hidden until answered, no moving on before answering, 2 hints, giving up, one similar question), Ask AI scope and memory, the lesson summary and mistake flashcards, memory and notes CRUD, speech limits and caching, the full loop, a failed attempt not unlocking anything, locked and coming-soon nodes, provider failures, a concurrent double-submit mastering once, all 7 lessons end to end, user isolation, the dashboard and streak, the v1 → v2 migration, and the auth flow (register, login, logout, expiry, validation, CSRF guard). Each test runs in its own schema.
- **`e2e.sh`** builds the API and runs it against a throwaway database with the mock tutor. It checks that anonymous users can't start lessons, then signs up two users. User A walks every ready lesson's steps and masters it over HTTP (with a failing checkpoint first), and user B must see none of it. It then restarts the API, logs in again, and checks that progress, flashcards and notes persisted.

## Languages (EN & ID)

The whole app works in **English and Bahasa Indonesia**.

- **Choosing a language:**
  - The first visit follows the device/browser language.
  - The **ID | EN** toggle is in the top nav, on the login/sign-up pages and on the Account page.
  - Once logged in, the choice is saved to the account (`users.locale`, via `PATCH /api/v1/auth/me`) and follows the user across devices.
- **UI text:** lives in `apps/mobile/src/i18n/en.ts` and `id.ts`. `id.ts` must have exactly the same keys as `en.ts`, so `tsc` fails if one is missing. API errors are translated from their `code`.
- **Course content:** the client sends `Accept-Language`, and the API returns course titles, levels, and roadmap node titles/summaries/objectives in that language. Translations live in `i18n` blocks inside the course files. A course that lists `"languages": ["en","id"]` must be fully translated, or the API refuses to start.
- **Tutor:** follows the language a lesson started in (`learning_sessions.locale`). With ID it explains and gives feedback mainly in Indonesian, while exercises and answers stay in English (`tutor.languagePolicy` in `course.json`). Switching the UI language mid-lesson doesn't change that lesson's tutor language.

## Step-by-step lessons

- **One thread per lesson.** A lesson shows one step at a time; finished steps fold into a row and can be reopened. The dashboard and roadmap stay the way in.
- **The steps are hand-written.** Every ready lesson has `packages/courses/<course>/plans/<nodeId>.json` with its steps in English and Indonesian (`{"en": {...}, "id": {...}}`), so a lesson opens instantly and every learner gets the same reviewed lesson. Step kinds: `concept` (card with word tiles), `examples` (with audio), `question` (multiple choice or word order), `timeline`, `flashcards`, `artifact` (cheat sheet) and `resources` (the lesson's curated links).
- **Checked at startup** (the API refuses to start otherwise): step kinds and count, every objective has a question, answer indexes in range, word-order tokens that rebuild the solution and come shuffled, no practice question repeats a checkpoint item, and the two languages match step for step with identical questions and answers. `ONLY_NODE=<id> go test ./internal/tutor -run TestAuthoredPlans` checks one lesson while writing it.
- **AI-written plans** remain as a fallback for lessons without hand-written steps (and for personalisation later): the AI writes 4–12 steps from the material, Go validates them (one retry), and otherwise a plan built from the material is used. These are cached for 7 days per lesson, course version, language, goal and memory.
- **Questions:** pick, then **Hint** (max 2, written by the AI; hints that contain the answer are rejected) or **Answer** (graded on the server). After both hints the learner may give up. After a wrong answer they choose **one similar question** ("3b", same rules, doesn't add to the step count) or continue. Answers and explanations stay on the server until a question is answered or given up.
- **Ask AI** opens inside explanation steps. The tutor answers questions about this lesson in full, answers questions about another lesson briefly and points to it, and politely declines anything outside English.
- **Checkpoint:** after the last step, one question at a time, typed or spoken, no hints. Passing writes a lesson summary (mastered · worth reviewing), saves the learner's mistakes as flashcards for Review, saves learning-pattern notes to memory, and prepares the next lesson's opening line. The next lesson starts with a chip of where the last one left off.
- **Memory** (Account → "What the tutor remembers about you"): onboarding answers, learning patterns and personal notes the tutor picked up from questions. Learners can edit or delete each note, delete all, or switch off new memories.
- **My notes:** cheat sheets saved from lessons. They can also be turned into flashcards.
- **Voice:** examples play with OpenAI text-to-speech, read one item at a time with the current word or sentence highlighted. **Lesson audio is committed to the repo** (`packages/courses/<course>/audio/<sha256 of the sentence>.mp3`), so a deployment serves it straight from the course folder and never calls the speech API for lessons. After adding or changing lesson steps, run `cd apps/api && go run ./cmd/server bundle-audio` (only new sentences are synthesised) and commit the new files; after changing `OPENAI_TTS_VOICE`/`OPENAI_TTS_MODEL`, `bundle-audio -fresh` re-records everything. The committed audio uses `gpt-4o-mini-tts` with the voice `alloy`, so keep those settings for new sentences to match; a test fails if any lesson sentence has no recording, and the API logs a warning at startup and makes the missing ones into `SPEECH_CACHE_DIR` (default `apps/api/data/tts`; `WARM_SPEECH=off` disables that). A call that is still running finishes and is cached even if the learner leaves, and the app uses the device voice (`expo-speech`) when the server has none or is slow (over 4 s). The mic in Ask AI and the checkpoint records with `expo-audio` and sends the clip to `/api/v1/transcribe`. Browsers only allow the mic on https or localhost.
- **Layout:**
  - Wide screens: lesson outline by level · the thread · cheat sheet or full material in a side panel.
  - Tablets: the thread with the panel beside it.
  - Phones: a compact header with step progress; cheat sheets and the material open as a sheet.
- **Flashcards:** saved cards follow a simple spaced-repetition schedule. *Again* brings a card back in 10 minutes; *Good* and *Easy* grow its interval by ×2.5 and ×3.5. They are reviewed on the **Review** page, and the dashboard shows how many are due.

### AI usage

- **Prompt caching:** every lesson call starts with the same prefix (policy → language → lesson material → scope rules → about the learner) and carries `prompt_cache_key = course:node:language`, so OpenAI bills the repeated prefix as cached input.
- **Exact-match grading:** checkpoint answers that exactly match a reference answer skip the evaluator, and practice questions are graded in Go without the AI.
- **Daily budget:** usage is logged for every call in `ai_usage`, and `AI_DAILY_TOKENS_PER_USER` (default 60,000; cached input doesn't count) caps it per learner per day. Over the cap, plans fall back to the material's plan and hints to built-in ones; Ask AI returns `ai_quota`.

## Personalization (onboarding) — switched off for now

> **Off by default.** Personalisation is paused while every lesson ships with hand-written steps. With it off there are no onboarding questions, goal badges, goal-based targets or tutor hints, and "Start this course" enrols directly. The code, course data and stored answers are kept: set `PERSONALIZATION=on` to bring it back (lessons then still use their hand-written steps; call `DropAuthoredPlans` to let the AI write personalised plans again).

Every learner gets the same lessons, objectives, assessments and pass mark. Onboarding only changes how the tutor teaches and which skills the roadmap highlights.

- **Questions:**
  - Each course defines its own in the `onboarding` block of `course.json`.
  - English asks about goal (required), background (with a free-text "something else"), self-reported level and learning style.
  - Each option carries:
    - Indonesian text;
    - an optional `targetLevel` (goal options);
    - an author-written `tutorHint` for the AI;
    - an optional `introLine` added to lesson intros.
- **Flow:**
  - "Start this course" opens `/onboarding/<course>`, which also enrols the learner.
  - Learners who enrolled earlier see a "Personalise your learning" card on the course page.
  - Skipping is recorded, so the card doesn't keep coming back.
  - Answers can be edited later from the course page.
- **Storage:** answers live on the enrollment (`enrollments.profile`, `onboarded_at`). They are read through `GET` and written through `PUT /api/v1/courses/{course}/profile`, and validated against the course's questions. Free text is capped at 80 characters and flattened to one line.
- **Roadmap:**
  - The goal sets the target level, and the learner can change it on the summary step.
  - Nodes list the goals they matter for in `focus` (in the level files). Matching nodes get a **For your goal** badge.
  - The next-lesson suggestion prefers them over other available nodes. Prerequisites still apply.
- **Tutor:**
  - The tutor prompt gains an "About the learner" section with the chosen options' hints. The learner's free text is quoted there as data, not instructions.
  - The evaluator gets the goal only to phrase feedback; scoring rules don't change.
  - The mock provider's plans don't change with the profile. The full effect needs the OpenAI provider.

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
- **No pronunciation scoring or voice roleplay yet.** Voice covers listening (TTS) and spoken answers (transcription).
- **AI plans vary.** Each learner's steps are written by the model; Go checks their structure, not their pedagogy, so an example can occasionally give away a later question.
- **Native video embeds:** on iOS/Android, videos open in YouTube rather than playing inline.
- **Android is only partly tested.** Web and the iOS simulator (Expo Go) were checked manually. Android needs an emulator/AVD, which wasn't available on the build machine.

## Next steps

1. Voice roleplay and more widgets (slider, chart).
2. Password reset and email verification.
3. More ready lessons (self-introduction, daily routines, everyday conversation).
4. Spaced repetition and a placement test.
5. Deploy: Go API + managed Postgres, and the Expo Web export (`npx expo export -p web`).
