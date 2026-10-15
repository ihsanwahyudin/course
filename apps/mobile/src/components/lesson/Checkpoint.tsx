import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Platform, StyleSheet, TextInput, View } from 'react-native';

import { LoadingState } from '@/components/ui/Bits';
import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { AssessmentResult, CheckItemView, Session } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useStartCheckpoint, useSubmitAssessment } from '@/lib/queries';
import { useUI } from '@/lib/store';
import { colors, fonts, radius } from '@/theme/tokens';
import { MicButton } from './AskBox';
import { AttemptHistory, AttemptReview } from './CheckpointReview';
import { ListenButton, Option, WordPicker } from './QuestionCard';
import { Indented, TutorLine } from './StepParts';

// Word-order drafts hold the picked token indexes ("2,0,1").
const pickedOf = (draft: string) => (draft ? draft.split(',').map(Number) : []);

// What gets sent: an mcq answer is the option's index, word order the sentence.
function answerOf(it: CheckItemView, draft: string) {
  if (it.type === 'word_order') return pickedOf(draft).map((i) => it.tokens?.[i] ?? '').join(' ');
  return draft.trim();
}

function filled(it: CheckItemView, draft: string) {
  if (it.type === 'word_order') return pickedOf(draft).length === (it.tokens?.length ?? 0);
  return draft.trim() !== '';
}

// The checkpoint: the only way to master a lesson, written fresh for every
// attempt. One question at a time, no hints; the AI grades and explains
// every answer and the server applies the pass mark. After mastery it runs
// as practice: recorded in the history, progress untouched.
export function Checkpoint({
  session,
  practice = false,
  onResult,
  onOpenMaterial,
  onProgress,
  onClose,
}: {
  session: Session;
  practice?: boolean;
  onResult: (r: AssessmentResult) => void;
  /** Lessons only; a level checkpoint has no material of its own. */
  onOpenMaterial?: () => void;
  onProgress?: () => void;
  onClose?: () => void;
}) {
  const t = useT();
  const router = useRouter();
  const errorText = useErrorText();
  const start = useStartCheckpoint(session.id);
  const lessonTitles = Object.fromEntries((session.level?.lessons ?? []).map((l) => [l.id, l.title]));
  const submit = useSubmitAssessment(session.id);
  const cp = session.checkpoint;
  const drafts = useUI((s) => (cp ? s.drafts[cp.id] : undefined)) ?? {};
  const setDraft = useUI((s) => s.setDraft);
  const clearDrafts = useUI((s) => s.clearDrafts);
  const [started, setStarted] = useState(false);
  const [idx, setIdx] = useState(0);
  // In practice mode, the result of the attempt just sent.
  const [practised, setPractised] = useState(false);
  // The learner's "message" that opened the questions.
  const [begunWith, setBegunWith] = useState('');
  const items = cp?.items ?? [];
  const item = items[idx];
  const last = session.lastAttempt;
  const failed = !practice && last && !last.passed ? last : null;
  const weak = failed ? session.node.objectives.filter((o) => (failed.objectiveScores[o.id] ?? 0) < failed.threshold) : [];
  // Level checkpoints: the lessons that went wrong, to go back to.
  const weakLessons = failed?.review?.lessons
    ? (session.level?.lessons ?? []).filter((l) => (failed.review!.lessons![l.id] ?? 100) < failed.threshold)
    : [];
  const objective = (id: string) => session.node.objectives.find((o) => o.id === id)?.text;
  const draft = item ? (drafts[item.id] ?? '') : '';
  const isLast = idx === items.length - 1;
  const answering = started && !!cp;
  const go = (i: number) => {
    setIdx(i);
    onProgress?.();
  };
  const set = (v: string) => cp && item && setDraft(cp.id, item.id, v);

  // Questions can be prepared before the learner opens them: "resume" only once they've answered something.
  const resuming = !!cp && Object.keys(drafts).length > 0;
  const startLabel = resuming ? t('checkpoint.resume') : failed || practised ? t('checkpoint.retryNew') : practice ? t('checkpoint.practiceStart') : t('checkpoint.start');
  const begin = () => {
    setBegunWith(startLabel);
    setPractised(false);
    setIdx(0);
    if (cp) {
      setStarted(true);
      onProgress?.();
      return;
    }
    start.mutate(undefined, {
      onSuccess: () => {
        setStarted(true);
        onProgress?.();
      },
    });
  };

  const send = () =>
    cp &&
    submit.mutate(
      { checkpointId: cp.id, answers: items.map((it) => ({ itemId: it.id, answer: answerOf(it, drafts[it.id] ?? '') })) },
      {
        onSuccess: (r) => {
          clearDrafts(cp.id);
          setStarted(false);
          setIdx(0);
          setPractised(practice);
          onResult(r);
          onProgress?.();
        },
      },
    );


  return (
    <View style={{ gap: 16 }}>
      {failed ? (
        <Indented style={{ paddingLeft: 0 }}>
          <View style={styles.failed} accessibilityLiveRegion="polite">
            <Text size={17} weight="medium" color={colors.danger}>
              {t('checkpoint.failedTitle', { score: failed.score })}
            </Text>
            <Text size={15}>{t('checkpoint.failedText', { threshold: failed.threshold })}</Text>
            {weak.map((o) => (
              <View key={o.id} style={{ flexDirection: 'row', gap: 8, alignItems: 'center' }}>
                <View style={styles.dot} />
                <Text size={15} style={{ flex: 1 }}>
                  {o.text}
                </Text>
              </View>
            ))}
            {weakLessons.length ? (
              <View style={{ gap: 8 }}>
                <Text size={15} weight="medium">
                  {t('level.weakTitle')}
                </Text>
                <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
                  {weakLessons.map((l) => (
                    <PillButton key={l.id} label={l.title} icon="arrowUpRight" size="sm" variant="secondary" onPress={() => router.push(`/learn/${session.courseSlug}/${l.id}`)} />
                  ))}
                </View>
              </View>
            ) : null}
            {failed.review ? <AttemptReview attempt={failed} lessonTitles={lessonTitles} /> : failed.feedback ? <Text size={14} color={colors.muted}>{failed.feedback}</Text> : null}
          </View>
        </Indented>
      ) : practised && last?.practice ? (
        <Indented style={{ paddingLeft: 0 }}>
          <View style={styles.practised} accessibilityLiveRegion="polite">
            <Text size={17} weight="medium">
              {t('checkpoint.practiceResult', { score: last.score })}
            </Text>
            <AttemptReview attempt={last} open lessonTitles={lessonTitles} />
          </View>
        </Indented>
      ) : !practice && !session.level ? (
        <TutorLine>{t('checkpoint.intro')}</TutorLine>
      ) : null}
      {/* After mastery the result card shows the history. */}
      {!practice && session.attempts.length > 1 && !answering ? (
        <Indented>
          <AttemptHistory attempts={session.attempts} current={last?.id} />
        </Indented>
      ) : null}

      {!answering ? (
        <Indented>
          <View style={styles.intro}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <Icon name="target" size={16} color={colors.white} />
              <Eyebrow color={colors.white}>
                {practice ? t('checkpoint.practiceTitle') : t('checkpoint.eyebrow')}
                {` · ${t('checkpoint.attemptN', { n: cp?.attempt ?? session.attempts.length + 1 })}`}
              </Eyebrow>
            </View>
            {start.isPending ? (
              <View style={{ gap: 6 }}>
                <LoadingState label={t('checkpoint.preparing')} />
                <Text size={14} color="rgba(255,255,255,0.72)">
                  {t('checkpoint.preparingText')}
                </Text>
              </View>
            ) : (
              <>
                <Heading size={24} color={colors.white}>
                  {cp ? t('checkpoint.title', { n: cp.items.length }) : t('checkpoint.titleNew')}
                </Heading>
                <Text size={14} color="rgba(255,255,255,0.72)">
                  {practice
                    ? t('checkpoint.practiceText')
                    : session.level
                      ? t('level.text', { threshold: session.node.masteryThreshold, level: session.node.level })
                      : t('checkpoint.text', { threshold: session.node.masteryThreshold, title: session.node.title })}
                </Text>
                <View style={{ flexDirection: 'row', gap: 10, flexWrap: 'wrap' }}>
                  <PillButton label={startLabel} variant="accent" onPress={begin} />
                  {practice && onClose ? (
                    <PillButton label={t('checkpoint.close')} onPress={onClose} style={{ backgroundColor: 'rgba(255,255,255,0.1)' }} />
                  ) : onOpenMaterial ? (
                    <PillButton label={failed ? t('checkpoint.openMaterial') : t('checkpoint.practise')} onPress={onOpenMaterial} style={{ backgroundColor: 'rgba(255,255,255,0.1)' }} />
                  ) : null}
                </View>
              </>
            )}
            {start.error ? (
              <Text size={14} color={colors.peach}>
                {errorText(start.error)}
              </Text>
            ) : null}
          </View>
        </Indented>
      ) : item && cp ? (
        <>
          <View style={styles.userBubble}>
            <Text size={15} color={colors.white}>
              {begunWith || startLabel}
            </Text>
          </View>
          <Indented>
            <View style={styles.card}>
              <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
                <Icon name="target" size={15} color={colors.accent} />
                <Eyebrow color={colors.accent} style={{ flex: 1 }}>
                  {practice ? t('checkpoint.practiceTitle') : t('checkpoint.noHints')}
                </Eyebrow>
                <Eyebrow>{t('checkpoint.of', { n: idx + 1, total: items.length })}</Eyebrow>
              </View>
              <View style={{ flexDirection: 'row', gap: 6 }} accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: items.length, now: idx + 1 }}>
                {items.map((it, i) => (
                  <View key={it.id} style={[styles.seg, { backgroundColor: i <= idx ? colors.accent : filled(it, drafts[it.id] ?? '') ? colors.lavender : colors.inkSoft }]} />
                ))}
              </View>
              <Text size={19} weight="medium" tight style={{ lineHeight: 28 }}>
                {item.prompt}
              </Text>
              {item.hasAudio ? <ListenButton key={item.id} path={`/api/v1/sessions/${session.id}/checkpoints/${cp.id}/items/${item.id}/listen`} /> : null}
              {item.lessonId && lessonTitles[item.lessonId] ? (
                <Text size={13} color={colors.muted}>
                  {t('level.from', { title: lessonTitles[item.lessonId] })}
                </Text>
              ) : objective(item.objectiveId) ? (
                <Text size={13} color={colors.muted}>
                  {t('checkpoint.objective', { text: objective(item.objectiveId)! })}
                </Text>
              ) : null}

              {item.type === 'mcq' ? (
                <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }} accessibilityRole="radiogroup">
                  {(item.options ?? []).map((o, i) => (
                    <Option key={`${item.id}-${i}`} label={o} state={draft === String(i) ? 'picked' : 'idle'} disabled={submit.isPending} onPress={() => set(String(i))} />
                  ))}
                </View>
              ) : item.type === 'word_order' ? (
                <WordPicker key={item.id} tokens={item.tokens ?? []} picked={pickedOf(draft)} setPicked={(p) => set(p.join(','))} />
              ) : (
                <View style={styles.inputRow}>
                  <TextInput
                    key={item.id}
                    value={draft}
                    onChangeText={set}
                    editable={!submit.isPending}
                    placeholder={t('step.yourAnswer')}
                    placeholderTextColor={colors.muted}
                    accessibilityLabel={t('checkpoint.answerLabel', { n: idx + 1, prompt: item.prompt })}
                    maxLength={1000}
                    autoFocus={Platform.OS === 'web'}
                    onSubmitEditing={() => (isLast ? filled(item, draft) && send() : filled(item, draft) && go(idx + 1))}
                    style={styles.input}
                  />
                  <MicButton onText={(txt) => set(draft ? `${draft} ${txt}` : txt)} />
                </View>
              )}

              <View style={{ flexDirection: 'row', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                <Text size={13} color={colors.muted} style={{ flex: 1, minWidth: 160 }}>
                  {item.type === 'mcq' ? t('checkpoint.pickOne') : item.type === 'word_order' ? t('checkpoint.buildSentence') : t('checkpoint.typeOrSpeak')}
                </Text>
                {idx > 0 ? <PillButton label={t('checkpoint.back')} variant="secondary" size="sm" disabled={submit.isPending} onPress={() => go(idx - 1)} /> : null}
                {isLast ? (
                  <PillButton
                    label={submit.isPending ? t('checkpoint.scoring') : t('checkpoint.submit')}
                    size="sm"
                    loading={submit.isPending}
                    disabled={items.some((it) => !filled(it, drafts[it.id] ?? ''))}
                    onPress={send}
                  />
                ) : (
                  <PillButton label={t('checkpoint.nextQuestion')} size="sm" disabled={!filled(item, draft)} onPress={() => go(idx + 1)} />
                )}
              </View>
              {submit.isPending ? (
                <Text size={13} color={colors.muted}>
                  {t('checkpoint.scoringText')}
                </Text>
              ) : null}
              {submit.error ? (
                <Text size={14} color={colors.danger} accessibilityLiveRegion="polite">
                  {errorText(submit.error, 'errors.submitFailed')}
                </Text>
              ) : null}
            </View>
          </Indented>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  intro: { backgroundColor: colors.ink, borderRadius: 24, padding: 22, gap: 12 },
  failed: { backgroundColor: colors.peach, borderRadius: 24, padding: 20, gap: 10 },
  practised: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: 24, padding: 20, gap: 12 },
  dot: { width: 7, height: 7, borderRadius: 4, backgroundColor: colors.danger },
  userBubble: { alignSelf: 'flex-end', backgroundColor: colors.ink, borderRadius: 18, borderBottomRightRadius: 6, paddingHorizontal: 16, paddingVertical: 11 },
  card: { backgroundColor: colors.card, borderWidth: 1.5, borderColor: colors.accent, borderRadius: 24, padding: 20, gap: 14 },
  seg: { flex: 1, height: 5, borderRadius: radius.pill },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: colors.canvas,
    borderWidth: 1,
    borderColor: colors.lavender,
    borderRadius: radius.xxl,
    paddingLeft: 14,
    paddingRight: 8,
    paddingVertical: 8,
  },
  input: {
    flex: 1,
    minHeight: 40,
    fontFamily: fonts.regular,
    fontSize: 17,
    color: colors.ink,
    ...(Platform.OS === 'web' ? ({ outlineStyle: 'none' } as object) : null),
  },
});
