import { QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { useLocale, currentLocale } from '@/i18n';

import { ApiError, api, type Intent, type ProfileInput, type ReviewGrade, type Session } from './api';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      retry: (count, err) => count < 2 && (!(err instanceof ApiError) || (err.retryable && err.status !== 401)),
    },
  },
});

// Server text is localized, so every key carries the UI language: switching
// language shows cached text for that language or refetches it.
export const keys = {
  courses: (l: string) => ['courses', l] as const,
  roadmap: (course: string, l: string) => ['roadmap', course, l] as const,
  node: (course: string, id: string, l: string) => ['node', course, id, l] as const,
  progress: (course: string, l: string) => ['progress', course, l] as const,
  dashboard: (l: string) => ['dashboard', l] as const,
  session: (id: string, l: string) => ['session', id, l] as const,
  profile: (course: string, l: string) => ['profile', course, l] as const,
  flashcards: (l: string) => ['flashcards', l] as const,
};

const useL = () => useLocale((s) => s.locale);

export const useCourses = () => {
  const l = useL();
  return useQuery({ queryKey: keys.courses(l), queryFn: api.courses, select: (d) => d.courses });
};
export const useRoadmap = (course: string) => {
  const l = useL();
  return useQuery({ queryKey: keys.roadmap(course, l), queryFn: () => api.roadmap(course), enabled: !!course });
};
export const useNode = (course: string, id: string) => {
  const l = useL();
  return useQuery({ queryKey: keys.node(course, id, l), queryFn: () => api.node(course, id), enabled: !!course && !!id });
};
export const useCourseProgress = (course: string | undefined) => {
  const l = useL();
  return useQuery({ queryKey: keys.progress(course ?? '', l), queryFn: () => api.courseProgress(course!), enabled: !!course });
};
export const useDashboard = (enabled = true) => {
  const l = useL();
  return useQuery({ queryKey: keys.dashboard(l), queryFn: api.dashboard, enabled });
};
export const useProfile = (course: string) => {
  const l = useL();
  return useQuery({ queryKey: keys.profile(course, l), queryFn: () => api.profile(course), enabled: !!course });
};
export const useSession = (id: string | undefined) => {
  const l = useL();
  return useQuery({ queryKey: keys.session(id ?? '', l), queryFn: () => api.session(id!), enabled: !!id });
};

// Anything that changes progress invalidates every progress-derived view.
function useInvalidateProgress() {
  const qc = useQueryClient();
  return () => {
    for (const k of [['courses'], ['roadmap'], ['node'], ['progress'], ['dashboard'], ['profile'], ['flashcards']]) {
      qc.invalidateQueries({ queryKey: k });
    }
  };
}

export function useEnroll() {
  const invalidate = useInvalidateProgress();
  return useMutation({ mutationFn: api.enroll, onSuccess: invalidate });
}

export const useDueFlashcards = () => {
  const l = useL();
  // Keep the queue stable while reviewing; refetch when the screen opens.
  return useQuery({ queryKey: keys.flashcards(l), queryFn: () => api.dueFlashcards(), staleTime: Infinity, refetchOnMount: 'always' });
};

export function useSaveFlashcards() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.saveFlashcards,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['flashcards'] });
      qc.invalidateQueries({ queryKey: ['dashboard'] });
    },
  });
}

export function useReviewFlashcard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, grade }: { id: string; grade: ReviewGrade }) => api.reviewFlashcard(id, grade),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dashboard'] }),
  });
}

export function useSaveProfile() {
  const invalidate = useInvalidateProgress();
  return useMutation({
    mutationFn: ({ course, input }: { course: string; input: ProfileInput }) => api.saveProfile(course, input),
    onSuccess: invalidate,
  });
}

export function useStartSession() {
  const qc = useQueryClient();
  const invalidate = useInvalidateProgress();
  return useMutation({
    mutationFn: ({ course, nodeId }: { course: string; nodeId: string }) => api.startSession(course, nodeId),
    onSuccess: (s: Session) => {
      qc.setQueryData(keys.session(s.id, currentLocale()), s);
      invalidate();
    },
  });
}

export function useSendMessage(sessionId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { content?: string; intent?: Intent }) => api.sendMessage(sessionId, b),
    onSuccess: (s) => {
      qc.setQueryData(keys.session(s.id, currentLocale()), s);
      qc.invalidateQueries({ queryKey: ['dashboard'] });
    },
  });
}

export function useSubmitAssessment(sessionId: string) {
  const qc = useQueryClient();
  const invalidate = useInvalidateProgress();
  return useMutation({
    mutationFn: (answers: { itemId: string; answer: string }[]) => api.submitAssessment(sessionId, answers),
    onSuccess: (r) => {
      qc.setQueryData(keys.session(r.session.id, currentLocale()), r.session);
      invalidate();
    },
  });
}
