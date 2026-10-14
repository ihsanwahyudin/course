import { create } from 'zustand';

// Ephemeral UI state (server state lives in TanStack Query).
type UIState = {
  roadmapView: 'graph' | 'list';
  setRoadmapView: (v: 'graph' | 'list') => void;
  // Unsent assessment answers, kept while the learner moves around the app.
  drafts: Record<string, Record<string, string>>;
  setDraft: (sessionId: string, itemId: string, value: string) => void;
  clearDrafts: (sessionId: string) => void;
};

export const useUI = create<UIState>((set) => ({
  roadmapView: 'graph',
  setRoadmapView: (roadmapView) => set({ roadmapView }),
  drafts: {},
  setDraft: (sessionId, itemId, value) =>
    set((s) => ({ drafts: { ...s.drafts, [sessionId]: { ...s.drafts[sessionId], [itemId]: value } } })),
  clearDrafts: (sessionId) =>
    set((s) => {
      const { [sessionId]: _, ...rest } = s.drafts;
      return { drafts: rest };
    }),
}));
