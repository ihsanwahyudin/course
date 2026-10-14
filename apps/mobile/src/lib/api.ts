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
export type Intent = 'ask' | 'simpler' | 'quiz' | 'flashcards' | 'examples' | 'video' | 'image';
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

export type User = { id: string; email: string; displayName: string; nativeLanguage: string; locale: 'en' | 'id'; createdAt: string };

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

export type Message = {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  phase: Phase;
  metadata: { kind?: string; intent?: Intent; quiz?: QuizQuestion[]; flashcards?: Flashcard[]; resources?: ResourceItem[]; cached?: boolean };
  createdAt: string;
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

export type QuizQuestion = { prompt: string; options: string[]; answer: number; explain: string };
export type ResourceItem = {
  kind: 'video' | 'article' | 'image';
  title: string;
  url: string;
  source: string;
  note?: string;
  videoId?: string;
  thumb?: string;
  author?: string;
  license?: string;
  curated: boolean;
};

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
  material: Material | null;
  greeting: string;
  messages: Message[];
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
  source: 'lesson' | 'ai';
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
  weekly: { messages: number; attempts: number; mastered: number };
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
  startSession: (course: string, nodeId: string) => request<Session>('/api/v1/sessions', post({ course, nodeId })),
  session: (id: string) => request<Session>(`/api/v1/sessions/${enc(id)}`),
  sendMessage: (id: string, b: { content?: string; intent?: Intent }) => request<Session>(`/api/v1/sessions/${enc(id)}/messages`, { ...post(b), timeoutMs: 95000 }),
  submitAssessment: (id: string, answers: { itemId: string; answer: string }[]) =>
    request<AssessmentResult>(`/api/v1/sessions/${enc(id)}/assessment`, { ...post({ answers }), timeoutMs: 95000 }),
  // flashcards
  saveFlashcards: (b: { course: string; nodeId: string; cards: Flashcard[] }) =>
    request<{ added: number; total: number; due: number }>('/api/v1/flashcards', post(b)),
  dueFlashcards: (course?: string) => request<ReviewQueue>(`/api/v1/flashcards/due${course ? `?course=${enc(course)}` : ''}`),
  reviewFlashcard: (id: string, grade: ReviewGrade) => request<SavedFlashcard>(`/api/v1/flashcards/${enc(id)}/review`, post({ grade })),
};
