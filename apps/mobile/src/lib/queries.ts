import { QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { ApiError, api, type Session } from './api';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      retry: (count, err) => count < 2 && (!(err instanceof ApiError) || (err.retryable && err.status !== 401)),
    },
  },
});

export const keys = {
  courses: ['courses'] as const,
  roadmap: (course: string) => ['roadmap', course] as const,
  node: (course: string, id: string) => ['node', course, id] as const,
  progress: (course: string) => ['progress', course] as const,
  dashboard: ['dashboard'] as const,
  session: (id: string) => ['session', id] as const,
};

export const useCourses = () => useQuery({ queryKey: keys.courses, queryFn: api.courses, select: (d) => d.courses });
export const useRoadmap = (course: string) => useQuery({ queryKey: keys.roadmap(course), queryFn: () => api.roadmap(course), enabled: !!course });
export const useNode = (course: string, id: string) => useQuery({ queryKey: keys.node(course, id), queryFn: () => api.node(course, id), enabled: !!course && !!id });
export const useCourseProgress = (course: string | undefined) =>
  useQuery({ queryKey: keys.progress(course ?? ''), queryFn: () => api.courseProgress(course!), enabled: !!course });
export const useDashboard = (enabled = true) => useQuery({ queryKey: keys.dashboard, queryFn: api.dashboard, enabled });

// Anything that changes progress invalidates every progress-derived view.
function useInvalidateProgress() {
  const qc = useQueryClient();
  return () => {
    for (const k of [['courses'], ['roadmap'], ['node'], ['progress'], ['dashboard']]) {
      qc.invalidateQueries({ queryKey: k });
    }
  };
}

export function useEnroll() {
  const invalidate = useInvalidateProgress();
  return useMutation({ mutationFn: api.enroll, onSuccess: invalidate });
}

export function useStartSession() {
  const qc = useQueryClient();
  const invalidate = useInvalidateProgress();
  return useMutation({
    mutationFn: ({ course, nodeId }: { course: string; nodeId: string }) => api.startSession(course, nodeId),
    onSuccess: (s: Session) => {
      qc.setQueryData(keys.session(s.id), s);
      invalidate();
    },
  });
}

export function useSendMessage(sessionId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (content: string) => api.sendMessage(sessionId, content),
    onSuccess: (s) => {
      qc.setQueryData(keys.session(s.id), s);
      qc.invalidateQueries({ queryKey: keys.dashboard });
    },
  });
}

export function useSubmitAssessment(sessionId: string) {
  const qc = useQueryClient();
  const invalidate = useInvalidateProgress();
  return useMutation({
    mutationFn: (answers: { itemId: string; answer: string }[]) => api.submitAssessment(sessionId, answers),
    onSuccess: (r) => {
      qc.setQueryData(keys.session(r.session.id), r.session);
      invalidate();
    },
  });
}
