import { useLocalSearchParams, useRouter } from 'expo-router';
import { useEffect, useRef, useState } from 'react';
import { KeyboardAvoidingView, Modal, Platform, ScrollView, StyleSheet, useWindowDimensions, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { Checkpoint } from '@/components/lesson/Checkpoint';
import { LessonResult } from '@/components/lesson/LessonResult';
import { LessonSidebar } from '@/components/lesson/LessonSidebar';
import { LessonThread } from '@/components/lesson/LessonThread';
import { LevelIntro } from '@/components/lesson/LevelIntro';
import { MaterialView } from '@/components/lesson/Material';
import { ArtifactActions, ArtifactView, artifactCards } from '@/components/lesson/Widgets';
import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import { ApiError, type Artifact, type RoadmapNode, type Session } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useRoadmap, useSaveFlashcards, useSaveNote, useSession, useStartSession } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';

const THREE_COLUMNS = 1100;
const INLINE_PANEL = 900;
const PANEL_WIDTH = 460;

type Panel = { kind: 'artifact'; artifact: Artifact } | { kind: 'material' };

// Which objectives have been practised, and which one the lesson is on.
function objectiveStates(s: Session) {
  const practised = new Set<string>();
  let current: string | undefined;
  for (const st of s.lesson?.steps ?? []) {
    for (const q of [st.question, st.similar]) {
      if (q && (q.answered || q.revealed)) practised.add(q.objectiveId);
      else if (q && !st.done) current = q.objectiveId;
    }
  }
  const mastered = s.phase === 'MASTERED';
  return s.node.objectives.map((o) => ({
    ...o,
    state: mastered || practised.has(o.id) ? ('done' as const) : o.id === (current ?? s.node.objectives.find((x) => !practised.has(x.id))?.id) ? ('current' as const) : ('todo' as const),
  }));
}

function ObjectiveChips({ s }: { s: Session }) {
  return (
    <View style={{ flexDirection: 'row', gap: 6, flexShrink: 1, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
      {objectiveStates(s).map((o) => (
        <View
          key={o.id}
          style={[
            styles.objective,
            o.state === 'done' && { backgroundColor: colors.mint, borderColor: colors.mint },
            o.state === 'current' && { backgroundColor: colors.accentSoft, borderColor: colors.accentSoft },
          ]}
          accessibilityLabel={o.text}
        >
          {o.state === 'done' ? <Icon name="check" size={12} color={colors.emeraldDeep} strokeWidth={2.5} /> : <View style={[styles.objDot, o.state === 'current' && { backgroundColor: colors.accent }]} />}
          <Text size={12} numberOfLines={1} color={o.state === 'done' ? colors.emeraldDeep : o.state === 'current' ? colors.accent : colors.muted} style={{ maxWidth: 170 }}>
            {o.text}
          </Text>
        </View>
      ))}
    </View>
  );
}

function useLessonNumber(course: string, node: RoadmapNode | undefined) {
  const { data } = useRoadmap(course);
  if (!data || !node) return undefined;
  return data.nodes.filter((n) => n.level === node.level).findIndex((n) => n.id === node.id) + 1;
}

// Side panel (desktop) or sheet (phone) with a cheat sheet or the full material.
function PanelContent({ panel, s, onClose }: { panel: Panel; s: Session; onClose: () => void }) {
  const t = useT();
  const saveNote = useSaveNote();
  const saveCards = useSaveFlashcards();
  const [noteSaved, setNoteSaved] = useState<string>();
  const [cardsMade, setCardsMade] = useState<string>();
  const [materialCards, setMaterialCards] = useState(false);
  const errorText = useErrorText();
  const a = panel.kind === 'artifact' ? panel.artifact : null;
  const err = saveNote.error ?? saveCards.error;
  return (
    <View style={{ flex: 1 }}>
      <View style={styles.panelHead}>
        <View style={{ flex: 1, gap: 2 }}>
          <Eyebrow>{a ? t('step.artifactEyebrow') : t('lesson.material')}</Eyebrow>
          <Text size={18} weight="medium" tight numberOfLines={2}>
            {a ? a.title : s.node.title}
          </Text>
        </View>
        <IconButton icon="close" label={t('common.close')} variant="soft" size={36} onPress={onClose} />
      </View>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 20 }}>
        {a ? (
          <ArtifactView artifact={a} />
        ) : s.material ? (
          <MaterialView
            material={s.material}
            savingCards={saveCards.isPending}
            cardsSaved={materialCards}
            onSaveCards={() => saveCards.mutate({ course: s.courseSlug, nodeId: s.nodeId, cards: s.material!.flashcards }, { onSuccess: () => setMaterialCards(true) })}
          />
        ) : null}
      </ScrollView>
      {a ? (
        <View style={styles.panelFoot}>
          {err ? (
            <Text size={13} color={colors.danger}>
              {errorText(err)}
            </Text>
          ) : null}
          <ArtifactActions
            noteSaved={noteSaved === a.title}
            cardsMade={cardsMade === a.title}
            saving={saveNote.isPending}
            making={saveCards.isPending}
            onSaveNote={() => saveNote.mutate({ course: s.courseSlug, nodeId: s.nodeId, artifact: a }, { onSuccess: () => setNoteSaved(a.title) })}
            onMakeCards={() => saveCards.mutate({ course: s.courseSlug, nodeId: s.nodeId, cards: artifactCards(a).slice(0, 30) }, { onSuccess: () => setCardsMade(a.title) })}
          />
        </View>
      ) : null}
    </View>
  );
}

// A lesson: one thread of steps shown one at a time, the checkpoint at the
// end, the course outline on wide screens and a panel for cheat sheets and
// the full material. Keyed by lesson so moving to the next one starts fresh.
export default function LearnScreen() {
  const { course, node: nodeId } = useLocalSearchParams<{ course: string; node: string }>();
  return <Lesson key={`${course}/${nodeId}`} course={course} nodeId={nodeId} />;
}

function Lesson({ course, nodeId }: { course: string; nodeId: string }) {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { width } = useWindowDimensions();
  const t = useT();
  const errorText = useErrorText();
  const start = useStartSession();
  const [panel, setPanel] = useState<Panel | null>(null);
  const [unlocked, setUnlocked] = useState<RoadmapNode[]>([]);
  const [practising, setPractising] = useState(false);
  const scrollRef = useRef<ScrollView>(null);

  const { mutate: startSession } = start;
  useEffect(() => {
    if (course && nodeId) startSession({ course, nodeId });
  }, [course, nodeId, startSession]);
  const sessionId = start.variables?.course === course && start.variables?.nodeId === nodeId ? start.data?.id : undefined;
  const session = useSession(sessionId);
  const s = session.data;
  const lessonNo = useLessonNumber(course, s?.node);

  const three = width >= THREE_COLUMNS;
  const wide = width >= 768;
  const inlinePanel = width >= INLINE_PANEL;
  const mainWidth = width - (three ? 300 : 0) - (panel && inlinePanel ? PANEL_WIDTH : 0);

  // Bring the newest step into view as the lesson moves on.
  const lesson = s?.lesson;
  const active = lesson?.steps[lesson.stepIndex];
  const progressKey = lesson
    ? [lesson.stepIndex, lesson.checkpoint, !!lesson.summary, s?.lastAttempt?.id, active?.question?.answered || active?.question?.revealed, !!active?.similar, active?.similar?.answered || active?.similar?.revealed].join('-')
    : '';
  useEffect(() => {
    if (!progressKey) return;
    const id = setTimeout(() => scrollRef.current?.scrollToEnd({ animated: true }), 120);
    return () => clearTimeout(id);
  }, [progressKey]);

  const toRoadmap = () => router.replace(`/courses/${course}`);
  const mastered = s?.phase === 'MASTERED';

  let main;
  if (start.isError) {
    const e = start.error;
    const blocked = e instanceof ApiError && (e.code === 'node_locked' || e.code === 'node_coming_soon');
    main = (
      <View style={styles.center}>
        <ErrorState title={blocked ? t('lesson.notOpen') : t('lesson.cantStart')} message={blocked ? e.message : errorText(e)} onRetry={blocked ? undefined : () => startSession({ course, nodeId })} />
        {blocked ? <PillButton label={t('common.backToRoadmap')} variant="secondary" onPress={toRoadmap} style={{ marginTop: 12 }} /> : null}
      </View>
    );
  } else if (!s) {
    main = (
      <View style={styles.center}>
        <View style={{ alignItems: 'center', gap: 4 }}>
          <LoadingState label={t('lesson.preparing')} />
          <Text size={14} color={colors.muted} style={{ textAlign: 'center', maxWidth: 360 }}>
            {t('lesson.preparingText')}
          </Text>
        </View>
      </View>
    );
  } else {
    const l = s.lesson;
    main = (
      <ScrollView ref={scrollRef} contentContainerStyle={[styles.scroll, { paddingBottom: insets.bottom + 80 }]} keyboardShouldPersistTaps="handled">
        <View style={styles.column}>
          {l ? <LessonThread session={s} openArtifact={(artifact) => setPanel({ kind: 'artifact', artifact })} /> : null}
          {s.level ? <LevelIntro session={s} /> : null}
          {(l || s.level) && mastered ? (
            <>
              <LessonResult
                session={s}
                unlocked={unlocked}
                onRoadmap={toRoadmap}
                onNext={(id) => router.replace(`/learn/${course}/${id}`)}
                onPractice={practising || s.checkpoint ? undefined : () => setPractising(true)}
              />
              {practising || s.checkpoint ? (
                <Checkpoint
                  session={s}
                  practice
                  // Keep the panel open to show the practice result.
                  onResult={() => setPractising(true)}
                  onOpenMaterial={s.material ? () => setPanel({ kind: 'material' }) : undefined}
                  onProgress={() => setTimeout(() => scrollRef.current?.scrollToEnd({ animated: true }), 80)}
                  onClose={() => setPractising(false)}
                />
              ) : null}
            </>
          ) : l?.checkpoint || s.level ? (
            <Checkpoint
              session={s}
              onResult={(r) => setUnlocked(r.unlocked)}
              onOpenMaterial={s.material ? () => setPanel({ kind: 'material' }) : undefined}
              onProgress={() => setTimeout(() => scrollRef.current?.scrollToEnd({ animated: true }), 80)}
            />
          ) : null}
          {!l && s.material ? <MaterialView material={s.material} /> : null}
        </View>
      </ScrollView>
    );
  }

  const stepText = !lesson ? '' : lesson.checkpoint ? t('checkpoint.eyebrow') : t('lesson.stepCount', { n: lesson.stepIndex + 1, total: lesson.totalSteps });
  const eyebrow = s ? [s.level ? `${s.node.level} · ${t('checkpoint.eyebrow')}` : lessonNo ? t('lesson.levelLesson', { level: s.node.level, n: lessonNo }) : s.node.level, mastered ? t('lesson.masteredTag') : null].filter(Boolean).join(' · ') : '';

  const header = s ? (
    wide ? (
      <View style={[styles.header, { paddingTop: three ? 0 : insets.top }]}>
        {!three ? <IconButton icon="arrowLeft" label={t('lesson.back')} variant="soft" size={36} onPress={toRoadmap} /> : null}
        <View style={{ flex: 1, minWidth: 140 }}>
          <Eyebrow numberOfLines={1}>{eyebrow}</Eyebrow>
          <Text size={18} weight="medium" tight numberOfLines={1}>
            {s.node.title}
          </Text>
        </View>
        {mainWidth >= 980 ? <ObjectiveChips s={s} /> : null}
        {lesson && !mastered ? (
          <View style={styles.stepPill}>
            <Eyebrow color={colors.ink}>{stepText}</Eyebrow>
          </View>
        ) : null}
        {s.material ? (
          <PillButton label={t('lesson.material')} icon="book" variant="secondary" size="sm" style={{ alignSelf: 'center' }} onPress={() => setPanel(panel?.kind === 'material' ? null : { kind: 'material' })} />
        ) : null}
      </View>
    ) : (
      <View style={[styles.phoneHeader, { paddingTop: insets.top + 8 }]}>
        <View style={{ flexDirection: 'row', alignItems: 'center', gap: 12 }}>
          <IconButton icon="arrowLeft" label={t('lesson.back')} variant="soft" size={36} onPress={toRoadmap} />
          <View style={{ flex: 1 }}>
            <Eyebrow numberOfLines={1}>{eyebrow}</Eyebrow>
            <Text size={17} weight="medium" tight numberOfLines={1}>
              {s.node.title}
            </Text>
          </View>
          {s.material ? <IconButton icon="book" label={t('lesson.material')} variant="soft" size={36} onPress={() => setPanel({ kind: 'material' })} /> : null}
        </View>
        {lesson ? (
          <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
            <View style={{ flex: 1, flexDirection: 'row', gap: 4 }} accessibilityRole="progressbar" accessibilityLabel={stepText}>
              {Array.from({ length: lesson.totalSteps }, (_, i) => (
                <View
                  key={i}
                  style={[styles.seg, { backgroundColor: mastered || i < lesson.stepIndex ? colors.emerald : i === lesson.stepIndex ? colors.accent : colors.inkSoft }]}
                />
              ))}
            </View>
            <Eyebrow>{`${Math.min(lesson.stepIndex + 1, lesson.totalSteps)}/${lesson.totalSteps}`}</Eyebrow>
          </View>
        ) : null}
      </View>
    )
  ) : null;

  return (
    <View style={styles.root}>
      <View style={{ flex: 1, flexDirection: 'row' }}>
        {three && course ? <LessonSidebar course={course} nodeId={nodeId} /> : null}
        <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
          {header}
          {main}
        </KeyboardAvoidingView>
        {s && panel && inlinePanel ? (
          <View style={[styles.side, { width: PANEL_WIDTH }]}>
            <PanelContent panel={panel} s={s} onClose={() => setPanel(null)} />
          </View>
        ) : null}
      </View>
      {s && panel && !inlinePanel ? (
        <Modal visible transparent animationType="slide" onRequestClose={() => setPanel(null)}>
          <View style={styles.backdrop}>
            <View style={[styles.sheet, { paddingBottom: insets.bottom, marginTop: insets.top + 40 }]}>
              <View style={styles.grabber} />
              <PanelContent panel={panel} s={s} onClose={() => setPanel(null)} />
            </View>
          </View>
        </Modal>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.canvas },
  center: { flex: 1, justifyContent: 'center', padding: 16 },
  scroll: { paddingHorizontal: 16, paddingTop: 28 },
  column: { width: '100%', maxWidth: 720, alignSelf: 'center', gap: 18 },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    minHeight: 72,
    paddingHorizontal: 24,
    borderBottomWidth: 1,
    borderBottomColor: colors.line,
    backgroundColor: colors.canvas,
  },
  phoneHeader: { gap: 12, paddingHorizontal: 16, paddingBottom: 12, borderBottomWidth: 1, borderBottomColor: colors.line, backgroundColor: colors.canvas },
  seg: { flex: 1, height: 4, borderRadius: radius.pill },
  stepPill: { borderWidth: 1, borderColor: colors.line, backgroundColor: colors.card, borderRadius: radius.pill, paddingHorizontal: 12, paddingVertical: 7 },
  objective: { flexDirection: 'row', alignItems: 'center', gap: 6, borderWidth: 1, borderColor: colors.line, backgroundColor: colors.card, borderRadius: radius.pill, paddingHorizontal: 10, paddingVertical: 5 },
  objDot: { width: 6, height: 6, borderRadius: 3, backgroundColor: colors.muted },
  side: { borderLeftWidth: 1, borderLeftColor: colors.line, backgroundColor: colors.card },
  panelHead: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingHorizontal: 20, paddingVertical: 14, borderBottomWidth: 1, borderBottomColor: colors.line },
  panelFoot: { padding: 16, gap: 8, borderTopWidth: 1, borderTopColor: colors.line },
  backdrop: { flex: 1, backgroundColor: 'rgba(17,17,19,0.35)', justifyContent: 'flex-end' },
  sheet: { flex: 1, backgroundColor: colors.card, borderTopLeftRadius: 28, borderTopRightRadius: 28, overflow: 'hidden' },
  grabber: { alignSelf: 'center', width: 40, height: 4, borderRadius: 2, backgroundColor: colors.lineStrong, marginTop: 10 },
});
