import { Platform } from 'react-native';

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
export type Phase = 'INTRODUCTION' | 'EXPLANATION' | 'GUIDED_PRACTICE' | 'INDEPENDENT_PRACTICE' | 'ASSESSMENT' | 'REVIEW' | 'MASTERED';
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
};

export type User = { id: string; email: string; displayName: string; nativeLanguage: string; createdAt: string };

export type CourseProgress = {
  percent: number;
  mastered: number;
  targetNodes: number;
  currentLevel: Level;
  targetLevel: Level;
  nextNode: RoadmapNode | null;
  lastActivityAt: string;
};

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
  metadata: { kind?: string; action?: string; observedMistakes?: string[]; advancedTo?: Phase };
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

export type Session = {
  id: string;
  courseSlug: string;
  nodeId: string;
  phase: Phase;
  phaseTurns: number;
  startedAt: string;
  updatedAt: string;
  endedAt: string | null;
  course: CourseRef;
  node: RoadmapNode;
  messages: Message[];
  steps: Phase[];
  canChat: boolean;
  assessment: { id: string; objectiveId: string; prompt: string }[];
  lastAttempt: Attempt | null;
};

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
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
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
  // courses
  courses: () => request<{ courses: Course[] }>('/api/v1/courses'),
  roadmap: (course: string) => request<Roadmap>(`/api/v1/courses/${enc(course)}/roadmap`),
  node: (course: string, id: string) => request<NodeDetail>(`/api/v1/courses/${enc(course)}/nodes/${enc(id)}`),
  enroll: (course: string) => request<Roadmap>(`/api/v1/courses/${enc(course)}/enroll`, post()),
  courseProgress: (course: string) => request<CourseProgressView>(`/api/v1/courses/${enc(course)}/progress`),
  dashboard: () => request<Dashboard>('/api/v1/dashboard'),
  // lessons
  startSession: (course: string, nodeId: string) => request<Session>('/api/v1/sessions', post({ course, nodeId })),
  session: (id: string) => request<Session>(`/api/v1/sessions/${enc(id)}`),
  sendMessage: (id: string, content: string) => request<Session>(`/api/v1/sessions/${enc(id)}/messages`, { ...post({ content }), timeoutMs: 95000 }),
  submitAssessment: (id: string, answers: { itemId: string; answer: string }[]) =>
    request<AssessmentResult>(`/api/v1/sessions/${enc(id)}/assessment`, { ...post({ answers }), timeoutMs: 95000 }),
};

export const phaseLabel: Record<Phase, string> = {
  INTRODUCTION: 'Intro',
  EXPLANATION: 'Explain',
  GUIDED_PRACTICE: 'Guided',
  INDEPENDENT_PRACTICE: 'Solo',
  ASSESSMENT: 'Assess',
  REVIEW: 'Review',
  MASTERED: 'Mastered',
};
