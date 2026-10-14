import { Platform } from 'react-native';

// Public, non-secret base URL of the Go API. AI keys live only on the server.
// Android emulators reach the host machine at 10.0.2.2; physical devices need
// EXPO_PUBLIC_API_URL set to the host's LAN IP or a tunnel.
const fallback = Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080';
export const API_URL = (process.env.EXPO_PUBLIC_API_URL || fallback).replace(/\/$/, '');

export type Level = 'A0' | 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';
export type NodeStatus = 'mastered' | 'in_progress' | 'available' | 'locked' | 'coming_soon';
export type Phase = 'INTRODUCTION' | 'EXPLANATION' | 'GUIDED_PRACTICE' | 'INDEPENDENT_PRACTICE' | 'ASSESSMENT' | 'REVIEW' | 'MASTERED';

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

export type Learner = {
  id: string;
  displayName: string;
  nativeLanguage: string;
  currentLevel: Level;
  targetLevel: Level;
  createdAt: string;
};

export type LevelSummary = { level: Level; title: string; description: string; total: number; ready: number; mastered: number };

export type Summary = {
  totalNodes: number;
  readyNodes: number;
  masteredNodes: number;
  targetNodes: number;
  targetMastered: number;
  levels: LevelSummary[];
};

export type Roadmap = {
  version: string;
  learner: Learner;
  nodes: RoadmapNode[];
  nextNodeId: string | null;
  summary: Summary;
};

export type NodeDetail = RoadmapNode & {
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
  nodeId: string;
  phase: Phase;
  phaseTurns: number;
  startedAt: string;
  updatedAt: string;
  endedAt: string | null;
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

export type Progress = {
  learner: Learner;
  summary: Summary;
  nextNode: RoadmapNode | null;
  mastered: { nodeId: string; title: string; level: Level; score: number | null; masteredAt: string }[];
  recentAttempts: (Attempt & { title: string; objectives: Objective[] })[];
  openSessions: { sessionId: string; nodeId: string; title: string; phase: Phase }[];
  weekly: { messages: number; attempts: number; mastered: number };
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

async function request<T>(path: string, init?: RequestInit & { timeoutMs?: number }): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), init?.timeoutMs ?? 20000);
  let res: Response;
  try {
    res = await fetch(API_URL + path, {
      ...init,
      signal: controller.signal,
      headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
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
    throw new ApiError(res.status, err?.code ?? 'http_error', err?.message ?? `Request failed (${res.status}).`, err?.retryable ?? res.status >= 500);
  }
  return body as T;
}

export const api = {
  roadmap: () => request<Roadmap>('/api/v1/roadmap'),
  progress: () => request<Progress>('/api/v1/progress'),
  node: (id: string) => request<NodeDetail>(`/api/v1/nodes/${encodeURIComponent(id)}`),
  startSession: (nodeId: string) => request<Session>('/api/v1/sessions', { method: 'POST', body: JSON.stringify({ nodeId }) }),
  session: (id: string) => request<Session>(`/api/v1/sessions/${encodeURIComponent(id)}`),
  sendMessage: (id: string, content: string) =>
    request<Session>(`/api/v1/sessions/${encodeURIComponent(id)}/messages`, { method: 'POST', body: JSON.stringify({ content }), timeoutMs: 95000 }),
  submitAssessment: (id: string, answers: { itemId: string; answer: string }[]) =>
    request<AssessmentResult>(`/api/v1/sessions/${encodeURIComponent(id)}/assessment`, { method: 'POST', body: JSON.stringify({ answers }), timeoutMs: 95000 }),
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
