import { useQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { Fragment, useEffect, useRef, useState } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { AssessmentForm } from '@/components/tutor/AssessmentForm';
import { ChatBubble, PhaseDivider, TypingBubble } from '@/components/tutor/ChatBubble';
import { Composer } from '@/components/tutor/Composer';
import { ResultPanel } from '@/components/tutor/ResultPanel';
import { ErrorState, LoadingState, pad2, PageDots } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern } from '@/components/ui/Decor';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { ApiError, api, phaseLabel, type AssessmentResult, type Phase, type Session } from '@/lib/api';
import { keys, useSendMessage, useStartSession, useSubmitAssessment } from '@/lib/queries';
import { useUI } from '@/lib/store';
import { colors } from '@/theme/tokens';

const suggestionsFor: Partial<Record<Phase, string[]>> = {
  INTRODUCTION: ["I'm ready!", 'Can you explain in Indonesian?'],
  EXPLANATION: ['Got it, next please', 'Give me another example', 'Jelaskan dalam bahasa Indonesia'],
  REVIEW: ["I'm ready to practise", 'Explain my mistakes again'],
};

function stepIndex(session: Session) {
  if (session.phase === 'REVIEW') return session.steps.indexOf('INDEPENDENT_PRACTICE');
  if (session.phase === 'MASTERED') return session.steps.length;
  return session.steps.indexOf(session.phase);
}

export default function LearnScreen() {
  const { course, node: nodeId } = useLocalSearchParams<{ course: string; node: string }>();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const start = useStartSession();
  const [draft, setDraft] = useState('');
  const [pendingText, setPendingText] = useState<string>();
  const [sendError, setSendError] = useState<{ text: string; message: string }>();
  const [result, setResult] = useState<AssessmentResult>();
  const scrollRef = useRef<ScrollView>(null);
  const clearDrafts = useUI((s) => s.clearDrafts);

  // Start or resume this node's session (the server makes this idempotent).
  const { mutate: startSession } = start;
  useEffect(() => {
    if (course && nodeId) startSession({ course, nodeId });
  }, [course, nodeId, startSession]);
  const sessionId = start.variables?.course === course && start.variables?.nodeId === nodeId ? start.data?.id : undefined;

  const session = useQuery({ queryKey: keys.session(sessionId ?? ''), queryFn: () => api.session(sessionId!), enabled: !!sessionId });
  const send = useSendMessage(sessionId ?? '');
  const submit = useSubmitAssessment(sessionId ?? '');
  const s = session.data;

  const doSend = (text: string) => {
    setSendError(undefined);
    setPendingText(text);
    setDraft('');
    send.mutate(text, {
      onSettled: () => setPendingText(undefined),
      onError: (e) => {
        setDraft(text);
        setSendError({ text, message: e instanceof ApiError ? e.message : 'Message failed to send.' });
      },
    });
  };

  const doSubmit = (answers: { itemId: string; answer: string }[]) => {
    submit.mutate(answers, {
      onSuccess: (r) => {
        clearDrafts(r.session.id);
        setResult(r);
      },
    });
  };

  useEffect(() => {
    const t = setTimeout(() => scrollRef.current?.scrollToEnd({ animated: true }), 60);
    return () => clearTimeout(t);
  }, [s?.messages.length, pendingText, s?.phase]);

  const header = (
    <View style={[styles.header, { paddingTop: insets.top + 12 }]}>
      <View style={styles.headerRow}>
        <IconButton icon="arrowLeft" label="Back to lesson details" variant="soft" onPress={() => (router.canGoBack() ? router.back() : router.replace(`/courses/${course}/${nodeId}`))} />
        <View style={{ flex: 1 }}>
          <Eyebrow>{s ? `${s.course.title} · ${s.node.level}` : 'Lesson'}</Eyebrow>
          <Text weight="medium" tight size={16} numberOfLines={1}>
            {s?.node.title ?? '…'}
          </Text>
        </View>
        {s ? (
          <View style={{ alignItems: 'flex-end', gap: 6 }}>
            <PageDots count={s.steps.length} active={stepIndex(s)} />
            <Eyebrow color={s.phase === 'REVIEW' ? colors.orange : colors.muted}>
              {s.phase === 'MASTERED' ? 'Complete' : `${pad2(Math.min(stepIndex(s) + 1, s.steps.length))}/${pad2(s.steps.length)} · ${phaseLabel[s.phase]}`}
            </Eyebrow>
          </View>
        ) : null}
      </View>
    </View>
  );

  let body;
  if (start.isError) {
    const e = start.error;
    const blocked = e instanceof ApiError && (e.code === 'node_locked' || e.code === 'node_coming_soon');
    body = (
      <View style={styles.center}>
        <ErrorState
          title={blocked ? 'This lesson isn’t open yet' : "Couldn't start the lesson"}
          message={e.message}
          onRetry={blocked ? undefined : () => startSession({ course, nodeId })}
        />
        {blocked ? <PillButton label="Back to roadmap" variant="secondary" onPress={() => router.replace(`/courses/${course}`)} style={{ marginTop: 12 }} /> : null}
      </View>
    );
  } else if (!s) {
    body = <LoadingState label="Opening lesson" />;
  } else {
    body = (
      <ScrollView ref={scrollRef} contentContainerStyle={styles.thread} keyboardShouldPersistTaps="handled">
        {s.messages.map((m, i) => {
          const prev = s.messages[i - 1];
          const phaseChanged = prev && prev.phase !== m.phase && m.phase !== 'MASTERED';
          return (
            <Fragment key={m.id}>
              {phaseChanged ? <PhaseDivider phase={m.phase} /> : null}
              <ChatBubble message={m} showAvatar={!prev || prev.role !== 'assistant' || phaseChanged} />
            </Fragment>
          );
        })}
        {pendingText ? (
          <>
            <ChatBubble message={{ role: 'user', content: pendingText, metadata: {} }} showAvatar={false} pending />
            <TypingBubble />
          </>
        ) : null}
        {sendError ? (
          <Card tone="peach" padding={16} style={{ gap: 10 }}>
            <Text size={14}>{sendError.message}</Text>
            <PillButton label="Retry" icon="refresh" size="sm" onPress={() => doSend(sendError.text)} />
          </Card>
        ) : null}
        {s.phase === 'ASSESSMENT' ? (
          <AssessmentForm session={s} onSubmit={doSubmit} submitting={submit.isPending} error={submit.error instanceof ApiError ? submit.error.message : submit.error ? 'Submission failed.' : undefined} />
        ) : null}
        {s.phase === 'MASTERED' && !result ? (
          <Card tone="mint" style={{ gap: 12 }}>
            <Eyebrow color="rgba(17,17,19,0.6)">Lesson complete</Eyebrow>
            <Text weight="medium" size={18} tight>
              {s.node.title} is mastered{s.lastAttempt ? ` with ${s.lastAttempt.score}/100` : ''}.
            </Text>
            <PillButton label="Back to roadmap" icon="arrowUpRight" onPress={() => router.replace(`/courses/${course}`)} />
          </Card>
        ) : null}
      </ScrollView>
    );
  }

  return (
    <View style={styles.root}>
      <DotPattern />
      <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={styles.column}>
          {header}
          <View style={{ flex: 1 }}>{body}</View>
          {s?.canChat ? (
            <View style={[styles.composer, { paddingBottom: insets.bottom + 14 }]}>
              <Composer value={draft} onChange={setDraft} onSend={doSend} disabled={send.isPending} suggestions={s.messages.length <= 6 || s.phase === 'REVIEW' ? suggestionsFor[s.phase] : undefined} />
            </View>
          ) : null}
        </View>
      </KeyboardAvoidingView>
      {result ? (
        <ResultPanel
          result={result}
          onClose={() => setResult(undefined)}
          onRoadmap={() => {
            setResult(undefined);
            router.replace(`/courses/${course}`);
          }}
          onNext={(id) => {
            setResult(undefined);
            router.replace(`/courses/${course}/${id}`);
          }}
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.canvas },
  column: { flex: 1, width: '100%', maxWidth: 760, alignSelf: 'center' },
  header: { paddingHorizontal: 16, paddingBottom: 12 },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    backgroundColor: 'rgba(255,255,255,0.8)',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 24,
    padding: 8,
    paddingRight: 16,
  },
  thread: { paddingHorizontal: 16, paddingVertical: 12, gap: 14 },
  composer: { paddingHorizontal: 16, paddingTop: 8 },
  center: { flex: 1, justifyContent: 'center', padding: 16 },
});
