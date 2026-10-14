import { QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { ApiError, api, type Session } from './api';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      retry: (count, err) => count < 2 && (!(err instanceof ApiError) || err.retryable),
    },
  },
});

export const keys = {
  roadmap: ['roadmap'] as const,
  progress: ['progress'] as const,
  node: (id: string) => ['node', id] as const,
  session: (id: string) => ['session', id] as const,
};

export const useRoadmap = () => useQuery({ queryKey: keys.roadmap, queryFn: api.roadmap });
export const useProgress = () => useQuery({ queryKey: keys.progress, queryFn: api.progress });
export const useNode = (id: string) => useQuery({ queryKey: keys.node(id), queryFn: () => api.node(id), enabled: !!id });

// Anything that changes progress invalidates the roadmap-derived views.
function useInvalidateProgress() {
  const qc = useQueryClient();
  return () => {
    qc.invalidateQueries({ queryKey: keys.roadmap });
    qc.invalidateQueries({ queryKey: keys.progress });
    qc.invalidateQueries({ queryKey: ['node'] });
  };
}

export function useStartSession() {
  const qc = useQueryClient();
  const invalidate = useInvalidateProgress();
  return useMutation({
    mutationFn: api.startSession,
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
    onSuccess: (s) => qc.setQueryData(keys.session(s.id), s),
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
