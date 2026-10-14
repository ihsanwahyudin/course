import { Platform } from 'react-native';

import { currentLocale } from '@/i18n';

// Public, non-secret base URL of the Go API. AI keys live only on the server.
// Android emulators reach the host machine at 10.0.2.2; physical devices need
// EXPO_PUBLIC_API_URL set to the host's LAN IP or a tunnel.
const fallback = Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080';

// On web, a localhost API URL only works on the dev machine itself. When the
// page is opened from another device on the LAN (e.g. http://192.168.x.x:8081),
// call the API on that same host instead.
function resolveApiUrl(configured: string) {
  if (Platform.OS !== 'web' || typeof window === 'undefined') return configured;
  const pageHost = window.location.hostname;
  try {
    const url = new URL(configured);
    const local = ['localhost', '127.0.0.1', '[::1]'];
    if (local.includes(url.hostname) && !local.includes(pageHost)) {
      url.hostname = pageHost;
      return url.toString();
    }
  } catch {
    // keep the configured value
  }
  return configured;
}

export const API_URL = resolveApiUrl(process.env.EXPO_PUBLIC_API_URL || fallback).replace(/\/$/, '');

// Levels are course data (course.json); English uses CEFR-style A0–C2.
export type Level = string;
export type NodeStatus = 'mastered' | 'in_progress' | 'available' | 'locked' | 'coming_soon';
export type Phase = 'LEARNING' | 'MASTERED';
export type Tone = 'default' | 'lavender' | 'peach' | 'mint' | 'sky';

export type Objective = { id: string; text: string };

export type RoadmapNode = {
  id: string;
  level: Level;
  title: string;
  summary: string;
  prerequisiteIds: string[];
  objectives: Objective[];
  masteryThreshold: number;
  contentStatus: 'ready' | 'coming_soon';
  status: NodeStatus;
  masteryScore: number | null;
  successorIds: string[];
  /** Tagged for the learner's onboarding goal. */
  recommended: boolean;
};

export type User = { id: string; email: string; displayName: string; nativeLanguage: string; locale: 'en' | 'id'; createdAt: string; memoryEnabled: boolean };

export type CourseProgress = {
  percent: number;
  mastered: number;
  targetNodes: number;
  currentLevel: Level;
  targetLevel: Level;
  nextNode: RoadmapNode | null;
  lastActivityAt: string;
  goal: Choice | null;
};

/** An onboarding answer as shown to the learner. */
export type Choice = { id: string; label: string };

export type ProfileSummary = {
  /** The course has onboarding questions. */
  available: boolean;
  /** Answered or skipped. */
  completed: boolean;
  goal: Choice | null;
  background: Choice | null;
};

export type OnboardingOption = { id: string; label: string; description?: string; targetLevel?: Level; freeText?: boolean };
export type OnboardingQuestion = { id: string; title: string; help?: string; required?: boolean; options: OnboardingOption[] };

export type ProfileView = {
  course: CourseRef;
  questions: OnboardingQuestion[];
  /** question id -> option id, plus "<question>_text" for free text. */
  answers: Record<string, string>;
  completed: boolean;
  targetLevel: Level;
  levels: Level[];
};

export type ProfileInput = { answers?: Record<string, string>; targetLevel?: Level; skip?: boolean };

export type Course = {
  slug: string;
  title: string;
  tagline: string;
  description: string;
  category: string;
  tone: Tone;
  status: 'published' | 'coming_soon';
  targetLevel: Level;
  levels: Level[];
  totalNodes: number;
  readyNodes: number;
  enrolled: boolean;
  progress: CourseProgress | null;
};

export type CourseRef = { slug: string; title: string };

export type LevelSummary = { level: Level; title: string; description: string; total: number; ready: number; mastered: number };

export type Summary = {
  totalNodes: number;
  readyNodes: number;
  masteredNodes: number;
  targetNodes: number;
  targetMastered: number;
  percent: number;
  levels: LevelSummary[];
};

export type Roadmap = {
  course: Course;
  authenticated: boolean;
  enrolled: boolean;
  enrollment: { currentLevel: Level; targetLevel: Level };
  version: string;
  nodes: RoadmapNode[];
  nextNodeId: string | null;
  summary: Summary;
  profile: ProfileSummary;
};

export type NodeDetail = RoadmapNode & {
  course: CourseRef;
  prerequisites: RoadmapNode[];
  successors: RoadmapNode[];
  canStart: boolean;
  blockedReason?: string;
  openSessionId: string | null;
  keyPoints: string[];
};

export type Attempt = {
  id: string;
  sessionId: string;
  courseSlug?: string;
  nodeId?: string;
  score: number;
  threshold: number;
  passed: boolean;
  objectiveScores: Record<string, number>;
  evidence: string[];
  feedback: string;
  createdAt: string;
};

// Lesson reading material (course data, localized by the server).
export type ExampleRow = { en: string; id: string; note?: string };
export type Block =
  | { type: 'text' | 'tip'; text: string }
  | { type: 'examples'; rows: ExampleRow[] }
  | { type: 'check'; prompt: string; options: string[]; answer: number; explain?: string };
export type Section = { id: string; title: string; blocks: Block[] };
export type CuratedResource = { kind: 'video' | 'article' | 'image'; title: string; url: string; source: string; note?: string };
export type Flashcard = { front: string; back: string };
export type Material = { lead: string; sections: Section[]; suggestedQuestions: string[]; resources: CuratedResource[]; flashcards: Flashcard[] };

// ---- step-by-step lessons ----

export type StepKind = 'concept' | 'examples' | 'question' | 'timeline' | 'flashcards' | 'artifact' | 'resources';
export type Pair = { en: string; id: string };
export type Artifact = { title: string; intro: string; headers: string[]; rows: string[][]; tips: string[]; examples: Pair[] };
export type Timeline = { title: string; caption: string; marks: { label: string; kind: 'past' | 'now' | 'future' }[] };

export type QuestionView = {
  type: 'mcq' | 'word_order';
  objectiveId: string;
  prompt: string;
  options: string[];
  tokens: string[];
  hints: string[];
  hintsLeft: number;
  choice: number | null;
  picked: string[] | null;
  answered: boolean;
  correct: boolean;
  revealed: boolean;
  /** Only once answered or revealed. */
  answer: number | null;
  solution: string;
  explanation: string;
};

export type CuratedLink = { kind: 'video' | 'article' | 'image'; title: string; url: string; source: string; note?: string; videoId?: string };
export type AskEntry = { question: string; answer: string; scope: 'in' | 'other_lesson' | 'off_topic'; otherLesson?: string };

export type StepView = {
  index: number;
  kind: StepKind;
  title: string;
  text: string;
  body: string;
  pairs: Pair[] | null;
  examples: Pair[] | null;
  timeline: Timeline | null;
  flashcards: Flashcard[] | null;
  artifact: Artifact | null;
  resources: CuratedLink[] | null;
  question: QuestionView | null;
  similar: QuestionView | null;
  asks: AskEntry[];
  done: boolean;
};

export type LessonSummary = {
  mastered: string[];
  review: string[];
  remembered: string[];
  flashcardsAdded: number;
  nextOpening: string;
  next: RoadmapNode | null;
};

export type Lesson = {
  greeting: string;
  steps: StepView[];
  stepIndex: number;
  totalSteps: number;
  /** Every step done: the checkpoint is next. */
  checkpoint: boolean;
  memoryChip: { from: string; strong: string; weak: string } | null;
  summary: LessonSummary | null;
};

export type AnswerInput = { choice?: number; tokens?: string[]; similar?: boolean };

export type MemoryItem = { id: string; courseSlug: string; kind: 'pattern' | 'personal'; text: string; sourceNode: string; sourceTitle: string; createdAt: string };
export type MemoryView = {
  enabled: boolean;
  profile: { course: string; courseTitle: string; question: string; answer: string }[];
  items: MemoryItem[];
};

export type Note = { id: string; courseSlug: string; nodeId: string; title: string; content: Artifact; createdAt: string; nodeTitle: string; courseTitle: string };

export type Session = {
  id: string;
  courseSlug: string;
  nodeId: string;
  phase: Phase;
  startedAt: string;
  updatedAt: string;
  endedAt: string | null;
  course: CourseRef;
  node: RoadmapNode;
  stepIndex: number;
  material: Material | null;
  lesson: Lesson | null;
  canChat: boolean;
  assessment: { id: string; objectiveId: string; prompt: string }[];
  lastAttempt: Attempt | null;
  prev: RoadmapNode | null;
  next: RoadmapNode | null;
};

export type SavedFlashcard = Flashcard & {
  id: string;
  courseSlug: string;
  nodeId: string;
  source: 'lesson' | 'ai' | 'mistake';
  dueAt: string;
  intervalDays: number;
  reps: number;
  lapses: number;
  nodeTitle: string;
  courseTitle: string;
};
export type ReviewQueue = { cards: SavedFlashcard[]; due: number; total: number };
export type ReviewGrade = 'again' | 'good' | 'easy';

export type AssessmentResult = {
  attempt: Attempt;
  passed: boolean;
  newlyMastered: boolean;
  alreadyCompleted: boolean;
  unlocked: RoadmapNode[];
  session: Session;
};

export type AttemptItem = Attempt & { title: string; courseTitle: string; objectives: Objective[] };

export type OpenSession = { sessionId: string; courseSlug: string; courseTitle: string; nodeId: string; title: string; phase: Phase; updatedAt: string };

export type CourseProgressView = {
  course: Course;
  summary: Summary;
  mastered: { nodeId: string; title: string; level: Level; score: number | null; masteredAt: string }[];
  recentAttempts: AttemptItem[];
  openSessions: OpenSession[];
};

export type Dashboard = {
  user: User;
  courses: Course[];
  explore: Course[];
  openSessions: OpenSession[];
  recentAttempts: AttemptItem[];
  weekly: { steps: number; attempts: number; mastered: number };
  streak: { current: number; activeToday: boolean };
  activity: { date: string; count: number }[];
  totalMastered: number;
  flashcards: { due: number; total: number };
};

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public retryable: boolean,
  ) {
    super(message);
  }
}

// Auth wiring, set by lib/auth: the bearer token (native only — web uses an
// HttpOnly cookie) and what to do when the server says the session is gone.
let bearerToken: string | null = null;
let onUnauthorized: (() => void) | null = null;
export function setAuthToken(token: string | null) {
  bearerToken = token;
}
export function setUnauthorizedHandler(fn: (() => void) | null) {
  onUnauthorized = fn;
}

/** Headers for requests made outside request() (audio downloads). */
export function authHeaders(): Record<string, string> {
  const h: Record<string, string> = { 'Accept-Language': currentLocale() };
  if (Platform.OS !== 'web') h['X-FluentPath-Client'] = 'native';
  if (bearerToken) h.Authorization = `Bearer ${bearerToken}`;
  return h;
}

async function request<T>(path: string, init?: RequestInit & { timeoutMs?: number; skipAuthRedirect?: boolean }): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), init?.timeoutMs ?? 20000);
  // The server localizes roadmap text and tutor messages from this header.
  const headers: Record<string, string> = { 'Content-Type': 'application/json', 'Accept-Language': currentLocale() };
  if (Platform.OS !== 'web') headers['X-FluentPath-Client'] = 'native';
  if (bearerToken) headers.Authorization = `Bearer ${bearerToken}`;
  let res: Response;
  try {
    res = await fetch(API_URL + path, {
      ...init,
      credentials: 'include',
      signal: controller.signal,
      headers: { ...headers, ...((init?.headers as Record<string, string>) ?? {}) },
    });
  } catch (e) {
    const aborted = (e as Error)?.name === 'AbortError';
    throw new ApiError(0, aborted ? 'timeout' : 'network', aborted ? 'The request timed out.' : `Can't reach the FluentPath server at ${API_URL}.`, true);
  } finally {
    clearTimeout(timer);
  }
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const err = body?.error;
    if (res.status === 401 && !init?.skipAuthRedirect) onUnauthorized?.();
    throw new ApiError(res.status, err?.code ?? 'http_error', err?.message ?? `Request failed (${res.status}).`, err?.retryable ?? res.status >= 500);
  }
  return body as T;
}

const post = (body?: unknown) => ({ method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) });
const enc = encodeURIComponent;

export const api = {
  // auth
  me: () => request<{ user: User }>('/api/v1/auth/me', { skipAuthRedirect: true }),
  register: (b: { email: string; password: string; displayName: string }) =>
    request<{ user: User; token?: string }>('/api/v1/auth/register', { ...post(b), skipAuthRedirect: true }),
  login: (b: { email: string; password: string }) => request<{ user: User; token?: string }>('/api/v1/auth/login', { ...post(b), skipAuthRedirect: true }),
  logout: () => request<{ ok: boolean }>('/api/v1/auth/logout', { ...post(), skipAuthRedirect: true }),
  updateMe: (b: { locale: 'en' | 'id' }) => request<{ user: User }>('/api/v1/auth/me', { method: 'PATCH', body: JSON.stringify(b) }),
  // courses
  courses: () => request<{ courses: Course[] }>('/api/v1/courses'),
  roadmap: (course: string) => request<Roadmap>(`/api/v1/courses/${enc(course)}/roadmap`),
  node: (course: string, id: string) => request<NodeDetail>(`/api/v1/courses/${enc(course)}/nodes/${enc(id)}`),
  enroll: (course: string) => request<Roadmap>(`/api/v1/courses/${enc(course)}/enroll`, post()),
  profile: (course: string) => request<ProfileView>(`/api/v1/courses/${enc(course)}/profile`),
  saveProfile: (course: string, b: ProfileInput) => request<Roadmap>(`/api/v1/courses/${enc(course)}/profile`, { method: 'PUT', body: JSON.stringify(b) }),
  courseProgress: (course: string) => request<CourseProgressView>(`/api/v1/courses/${enc(course)}/progress`),
  dashboard: () => request<Dashboard>('/api/v1/dashboard'),
  // lessons
  // The first visit writes the lesson plan with the AI, which can take a while.
  startSession: (course: string, nodeId: string) => request<Session>('/api/v1/sessions', { ...post({ course, nodeId }), timeoutMs: 170000 }),
  session: (id: string) => request<Session>(`/api/v1/sessions/${enc(id)}`),
  step: (id: string, n: number, action: 'next' | 'hint' | 'reveal' | 'similar', b?: { similar?: boolean }) =>
    request<Session>(`/api/v1/sessions/${enc(id)}/steps/${n}/${action}`, { ...post(b), timeoutMs: 95000 }),
  answer: (id: string, n: number, b: AnswerInput) => request<Session>(`/api/v1/sessions/${enc(id)}/steps/${n}/answer`, post(b)),
  ask: (id: string, n: number, content: string) => request<Session>(`/api/v1/sessions/${enc(id)}/steps/${n}/ask`, { ...post({ content }), timeoutMs: 95000 }),
  // speech
  transcribe: (audio: Blob, type: string) =>
    request<{ text: string }>('/api/v1/transcribe', { method: 'POST', body: audio, headers: { 'Content-Type': type }, timeoutMs: 60000 }),
  // memory & notes
  memory: () => request<MemoryView>('/api/v1/me/memory'),
  setMemoryEnabled: (enabled: boolean) => request<MemoryView>('/api/v1/me/memory', { method: 'PATCH', body: JSON.stringify({ enabled }) }),
  updateMemory: (id: string, text: string) => request<MemoryView>(`/api/v1/me/memory/${enc(id)}`, { method: 'PATCH', body: JSON.stringify({ text }) }),
  deleteMemory: (id: string) => request<MemoryView>(`/api/v1/me/memory/${enc(id)}`, { method: 'DELETE' }),
  deleteAllMemory: () => request<MemoryView>('/api/v1/me/memory', { method: 'DELETE' }),
  notes: () => request<{ notes: Note[] }>('/api/v1/notes'),
  saveNote: (b: { course: string; nodeId: string; artifact: Artifact }) => request<Note>('/api/v1/notes', post(b)),
  deleteNote: (id: string) => request<{ ok: boolean }>(`/api/v1/notes/${enc(id)}`, { method: 'DELETE' }),
  submitAssessment: (id: string, answers: { itemId: string; answer: string }[]) =>
    request<AssessmentResult>(`/api/v1/sessions/${enc(id)}/assessment`, { ...post({ answers }), timeoutMs: 95000 }),
  // flashcards
  saveFlashcards: (b: { course: string; nodeId: string; cards: Flashcard[] }) =>
    request<{ added: number; total: number; due: number }>('/api/v1/flashcards', post(b)),
  dueFlashcards: (course?: string) => request<ReviewQueue>(`/api/v1/flashcards/due${course ? `?course=${enc(course)}` : ''}`),
  reviewFlashcard: (id: string, grade: ReviewGrade) => request<SavedFlashcard>(`/api/v1/flashcards/${enc(id)}/review`, post({ grade })),
};
