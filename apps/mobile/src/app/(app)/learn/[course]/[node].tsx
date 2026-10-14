import { useLocalSearchParams, useRouter } from 'expo-router';
import { useEffect, useState } from 'react';
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, useWindowDimensions, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { LessonSidebar } from '@/components/lesson/LessonSidebar';
import { MaterialView } from '@/components/lesson/Material';
import { AssessmentForm } from '@/components/tutor/AssessmentForm';
import { HelperPanel } from '@/components/tutor/HelperPanel';
import { ResultPanel } from '@/components/tutor/ResultPanel';
import { ErrorState, LoadingState, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import { ApiError, type AssessmentResult, type RoadmapNode, type Session } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useSaveFlashcards, useSession, useStartSession, useSubmitAssessment } from '@/lib/queries';
import { useUI } from '@/lib/store';
import { colors, radius, shadow } from '@/theme/tokens';

const THREE_COLUMNS = 1100;
const TWO_COLUMNS = 768;

function NavButton({ node, dir, onPress }: { node: RoadmapNode | null; dir: 'prev' | 'next'; onPress: () => void }) {
  const t = useT();
  if (!node) return <View style={{ flex: 1 }} />;
  const locked = node.status === 'locked';
  return (
    <Pressable onPress={onPress} accessibilityRole="link" style={[styles.nav, dir === 'next' && { alignItems: 'flex-end' }]}>
      <Eyebrow>
        {dir === 'prev' ? `‹ ${t('lesson.prev')}` : `${t('lesson.next')} ›`}
        {locked ? ` · ${t('lesson.locked')}` : ''}
      </Eyebrow>
      <Text weight="medium" tight numberOfLines={1} color={locked ? colors.muted : colors.ink}>
        {node.title}
      </Text>
    </Pressable>
  );
}

// The final quiz card at the end of the material.
function FinalQuizCard({ s, onOpen }: { s: Session; onOpen: () => void }) {
  const t = useT();
  const last = s.lastAttempt;
  const weak = last && !last.passed ? s.node.objectives.filter((o) => (last.objectiveScores[o.id] ?? 0) < last.threshold) : [];
  if (s.phase === 'MASTERED') {
    return (
      <Card tone="mint" style={{ gap: 10 }}>
        <Eyebrow color={colors.emeraldDeep}>{t('lesson.complete')}</Eyebrow>
        <Text weight="medium" size={18} tight>
          {last ? t('lesson.masteredScore', { title: s.node.title, score: last.score }) : `${t('lesson.masteredLine', { title: s.node.title })}.`}
        </Text>
      </Card>
    );
  }
  return (
    <Card style={{ gap: 12 }}>
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
        <Icon name="target" size={18} color={colors.accent} />
        <Eyebrow color={colors.accent}>{t('lesson.finalQuiz')}</Eyebrow>
        {last ? <Eyebrow style={{ marginLeft: 'auto' }}>{t('lesson.lastScore', { score: last.score })}</Eyebrow> : null}
      </View>
      <Text color={colors.muted}>{t('lesson.finalQuizText', { n: s.assessment.length, threshold: s.node.masteryThreshold })}</Text>
      {weak.length ? (
        <Text size={14} color={colors.danger}>
          {t('lesson.reviewSections', { objectives: weak.map((o) => o.text).join('; ') })}
        </Text>
      ) : null}
      <PillButton label={last ? t('lesson.retakeQuiz') : t('lesson.startQuiz')} icon="arrowUpRight" onPress={onOpen} style={{ alignSelf: 'flex-start' }} />
    </Card>
  );
}

// A lesson: reading material in the middle, the AI helper beside it (or one
// tap away on phones), the course outline on wide screens, and the final quiz
// that decides mastery.
export default function LearnScreen() {
  const { course, node: nodeId } = useLocalSearchParams<{ course: string; node: string }>();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { width } = useWindowDimensions();
  const t = useT();
  const errorText = useErrorText();
  const start = useStartSession();
  const [quizOpen, setQuizOpen] = useState(false);
  const [helperOpen, setHelperOpen] = useState(false);
  const [result, setResult] = useState<AssessmentResult>();
  const [cardsSaved, setCardsSaved] = useState<string>();
  const clearDrafts = useUI((s) => s.clearDrafts);
  const saveCards = useSaveFlashcards();

  // Start or resume this node's session (the server makes this idempotent).
  const { mutate: startSession } = start;
  useEffect(() => {
    if (course && nodeId) startSession({ course, nodeId });
  }, [course, nodeId, startSession]);
  const sessionId = start.variables?.course === course && start.variables?.nodeId === nodeId ? start.data?.id : undefined;
  const session = useSession(sessionId);
  const submit = useSubmitAssessment(sessionId ?? '');
  const s = session.data;

  const three = width >= THREE_COLUMNS;
  const two = width >= TWO_COLUMNS;
  const goTo = (n: RoadmapNode) => router.replace(n.status === 'locked' ? `/courses/${course}/${n.id}` : `/learn/${course}/${n.id}`);

  const doSubmit = (answers: { itemId: string; answer: string }[]) => {
    submit.mutate(answers, {
      onSuccess: (r) => {
        clearDrafts(r.session.id);
        setQuizOpen(false);
        setResult(r);
      },
    });
  };

  let main;
  if (start.isError) {
    const e = start.error;
    const blocked = e instanceof ApiError && (e.code === 'node_locked' || e.code === 'node_coming_soon');
    main = (
      <View style={styles.center}>
        <ErrorState
          title={blocked ? t('lesson.notOpen') : t('lesson.cantStart')}
          message={blocked ? e.message : errorText(e)}
          onRetry={blocked ? undefined : () => startSession({ course, nodeId })}
        />
        {blocked ? <PillButton label={t('common.backToRoadmap')} variant="secondary" onPress={() => router.replace(`/courses/${course}`)} style={{ marginTop: 12 }} /> : null}
      </View>
    );
  } else if (!s) {
    main = (
      <View style={styles.center}>
        <LoadingState label={t('lesson.opening')} />
      </View>
    );
  } else {
    main = (
      <ScrollView contentContainerStyle={[styles.materialScroll, { paddingTop: three ? 40 : insets.top + 16, paddingBottom: two ? 64 : insets.bottom + 150 }]}>
        <View style={styles.material}>
          {!three ? (
            <Pressable onPress={() => router.replace(`/courses/${course}`)} accessibilityRole="link" style={styles.crumb}>
              <Icon name="arrowLeft" size={16} color={colors.muted} />
              <Text size={14} color={colors.muted} numberOfLines={1}>
                {s.course.title}
              </Text>
            </Pressable>
          ) : null}
          <View style={{ gap: 12, marginBottom: 28 }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Eyebrow>
                {s.course.title} · {s.node.level}
              </Eyebrow>
              <StatusChip status={s.node.status} />
            </View>
            <Heading size={two ? 52 : 38}>{s.node.title}</Heading>
          </View>
          {s.material ? (
            <MaterialView
              material={s.material}
              savingCards={saveCards.isPending}
              cardsSaved={cardsSaved === s.id}
              onSaveCards={() => saveCards.mutate({ course, nodeId, cards: s.material!.flashcards }, { onSuccess: () => setCardsSaved(s.id) })}
            />
          ) : null}
          <View style={{ marginTop: 44, gap: 16 }}>
            <FinalQuizCard s={s} onOpen={() => setQuizOpen(true)} />
            {two ? (
              <View style={styles.navRow}>
                <NavButton node={s.prev} dir="prev" onPress={() => s.prev && goTo(s.prev)} />
                <NavButton node={s.next} dir="next" onPress={() => s.next && goTo(s.next)} />
              </View>
            ) : null}
          </View>
        </View>
      </ScrollView>
    );
  }

  return (
    <View style={styles.root}>
      <View style={{ flex: 1, flexDirection: 'row' }}>
        {three && course ? <LessonSidebar course={course} nodeId={nodeId} /> : null}
        <View style={{ flex: 1 }}>{main}</View>
        {two && s ? (
          <View style={[styles.side, { width: three ? 380 : 340, paddingTop: three ? 0 : insets.top }]}>
            <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
              <HelperPanel session={s} />
            </KeyboardAvoidingView>
          </View>
        ) : null}
      </View>

      {/* Phones: bottom bar + floating "Ask AI" that opens the helper full-screen. */}
      {!two && s ? (
        <>
          <Pressable onPress={() => setHelperOpen(true)} accessibilityRole="button" style={[styles.fab, { bottom: insets.bottom + 84, boxShadow: shadow.cardHover }]}>
            <Icon name="sparkle" size={18} color={colors.white} />
            <Text size={14} weight="medium" color={colors.white}>
              {t('lesson.askAi')}
            </Text>
          </Pressable>
          <View style={[styles.bar, { paddingBottom: insets.bottom + 10 }]}>
            <IconButton icon="arrowLeft" label={t('lesson.prev')} variant="soft" onPress={() => (s.prev ? goTo(s.prev) : router.replace(`/courses/${course}`))} />
            {s.phase !== 'MASTERED' ? (
              <PillButton label={t('lesson.finalQuiz')} icon="target" onPress={() => setQuizOpen(true)} style={{ flex: 1 }} />
            ) : (
              <View style={{ flex: 1, alignItems: 'center' }}>
                <Eyebrow color={colors.emeraldDeep}>{t('lesson.complete')}</Eyebrow>
              </View>
            )}
            <IconButton icon="arrowRight" label={t('lesson.next')} variant="soft" onPress={() => (s.next ? goTo(s.next) : router.replace(`/courses/${course}`))} />
          </View>
          <Modal visible={helperOpen} animationType="slide" onRequestClose={() => setHelperOpen(false)}>
            <KeyboardAvoidingView style={{ flex: 1, paddingTop: insets.top, paddingBottom: insets.bottom, backgroundColor: colors.card }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
              <HelperPanel session={s} onClose={() => setHelperOpen(false)} />
            </KeyboardAvoidingView>
          </Modal>
        </>
      ) : null}

      {s && quizOpen ? (
        <Modal visible transparent animationType="fade" onRequestClose={() => setQuizOpen(false)}>
          <View style={styles.backdrop}>
            <ScrollView contentContainerStyle={[styles.quizScroll, { paddingTop: insets.top + 24, paddingBottom: insets.bottom + 24 }]} keyboardShouldPersistTaps="handled">
              <View style={{ width: '100%', maxWidth: 640, gap: 10 }}>
                <View style={{ alignItems: 'flex-end' }}>
                  <IconButton icon="close" label={t('common.close')} variant="soft" onPress={() => setQuizOpen(false)} />
                </View>
                <AssessmentForm session={s} onSubmit={doSubmit} submitting={submit.isPending} error={submit.error ? errorText(submit.error, 'errors.submitFailed') : undefined} />
              </View>
            </ScrollView>
          </View>
        </Modal>
      ) : null}

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
            router.replace(`/learn/${course}/${id}`);
          }}
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.canvas },
  center: { flex: 1, justifyContent: 'center', padding: 16 },
  materialScroll: { paddingHorizontal: 20 },
  material: { width: '100%', maxWidth: 720, alignSelf: 'center' },
  crumb: { flexDirection: 'row', alignItems: 'center', gap: 6, marginBottom: 20, alignSelf: 'flex-start' },
  side: { borderLeftWidth: 1, borderLeftColor: colors.line, backgroundColor: colors.card },
  navRow: { flexDirection: 'row', gap: 12 },
  nav: { flex: 1, gap: 4, borderWidth: 1, borderColor: colors.line, borderRadius: 20, padding: 16, backgroundColor: colors.card },
  fab: {
    position: 'absolute',
    right: 16,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: colors.accent,
    borderRadius: radius.pill,
    paddingHorizontal: 18,
    paddingVertical: 12,
  },
  bar: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    paddingHorizontal: 16,
    paddingTop: 10,
    backgroundColor: colors.card,
    borderTopWidth: 1,
    borderTopColor: colors.line,
  },
  backdrop: { flex: 1, backgroundColor: 'rgba(17,17,19,0.35)' },
  quizScroll: { alignItems: 'center', paddingHorizontal: 16 },
});
