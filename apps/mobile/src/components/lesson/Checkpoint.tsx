import { useState } from 'react';
import { Platform, StyleSheet, TextInput, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { AssessmentResult, Session } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useSubmitAssessment } from '@/lib/queries';
import { useUI } from '@/lib/store';
import { colors, fonts, radius } from '@/theme/tokens';
import { MicButton } from './AskBox';
import { Indented, TutorLine } from './StepParts';

// The checkpoint: the only way to master a lesson. One question at a time,
// no hints, answered by typing or speaking; the server scores it.
export function Checkpoint({ session, onResult, onOpenMaterial, onProgress }: { session: Session; onResult: (r: AssessmentResult) => void; onOpenMaterial: () => void; onProgress?: () => void }) {
  const t = useT();
  const errorText = useErrorText();
  const submit = useSubmitAssessment(session.id);
  const drafts = useUI((s) => s.drafts[session.id]) ?? {};
  const setDraft = useUI((s) => s.setDraft);
  const clearDrafts = useUI((s) => s.clearDrafts);
  const [started, setStarted] = useState(false);
  const [idx, setIdx] = useState(0);
  const items = session.assessment;
  const item = items[idx];
  const last = session.lastAttempt;
  const failed = last && !last.passed ? last : null;
  const weak = failed ? session.node.objectives.filter((o) => (failed.objectiveScores[o.id] ?? 0) < failed.threshold) : [];
  const objective = (id: string) => session.node.objectives.find((o) => o.id === id)?.text;
  const value = item ? (drafts[item.id] ?? '') : '';
  const isLast = idx === items.length - 1;
  const go = (i: number) => {
    setIdx(i);
    onProgress?.();
  };

  const send = () =>
    submit.mutate(
      items.map((it) => ({ itemId: it.id, answer: (drafts[it.id] ?? '').trim() })),
      {
        onSuccess: (r) => {
          clearDrafts(session.id);
          setStarted(false);
          setIdx(0);
          onResult(r);
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
            {failed.feedback ? (
              <Text size={14} color={colors.muted}>
                {failed.feedback}
              </Text>
            ) : null}
          </View>
        </Indented>
      ) : (
        <TutorLine>{t('checkpoint.intro')}</TutorLine>
      )}

      {!started ? (
        <Indented>
          <View style={styles.intro}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <Icon name="target" size={16} color={colors.white} />
              <Eyebrow color={colors.white}>{t('checkpoint.eyebrow')}</Eyebrow>
            </View>
            <Heading size={24} color={colors.white}>
              {t('checkpoint.title', { n: items.length })}
            </Heading>
            <Text size={14} color="rgba(255,255,255,0.72)">
              {t('checkpoint.text', { threshold: session.node.masteryThreshold, title: session.node.title })}
            </Text>
            <View style={{ flexDirection: 'row', gap: 10, flexWrap: 'wrap' }}>
              <PillButton label={failed ? t('checkpoint.retry') : t('checkpoint.start')} variant="accent" onPress={() => (setStarted(true), onProgress?.())} />
              <PillButton label={failed ? t('checkpoint.openMaterial') : t('checkpoint.practise')} onPress={onOpenMaterial} style={{ backgroundColor: 'rgba(255,255,255,0.1)' }} />
            </View>
          </View>
        </Indented>
      ) : item ? (
        <>
          <View style={styles.userBubble}>
            <Text size={15} color={colors.white}>
              {failed ? t('checkpoint.retry') : t('checkpoint.start')}
            </Text>
          </View>
          <Indented>
            <View style={styles.card}>
              <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
                <Icon name="target" size={15} color={colors.accent} />
                <Eyebrow color={colors.accent} style={{ flex: 1 }}>
                  {t('checkpoint.noHints')}
                </Eyebrow>
                <Eyebrow>{t('checkpoint.of', { n: idx + 1, total: items.length })}</Eyebrow>
              </View>
              <View style={{ flexDirection: 'row', gap: 6 }} accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: items.length, now: idx + 1 }}>
                {items.map((it, i) => (
                  <View key={it.id} style={[styles.seg, { backgroundColor: i <= idx ? colors.accent : (drafts[it.id] ?? '').trim() ? colors.lavender : colors.inkSoft }]} />
                ))}
              </View>
              <Text size={19} weight="medium" tight style={{ lineHeight: 28 }}>
                {item.prompt}
              </Text>
              {objective(item.objectiveId) ? (
                <Text size={13} color={colors.muted}>
                  {t('checkpoint.objective', { text: objective(item.objectiveId)! })}
                </Text>
              ) : null}
              <View style={styles.inputRow}>
                <TextInput
                  key={item.id}
                  value={value}
                  onChangeText={(v) => setDraft(session.id, item.id, v)}
                  editable={!submit.isPending}
                  placeholder={t('step.yourAnswer')}
                  placeholderTextColor={colors.muted}
                  accessibilityLabel={t('checkpoint.answerLabel', { n: idx + 1, prompt: item.prompt })}
                  maxLength={1000}
                  autoFocus={Platform.OS === 'web'}
                  onSubmitEditing={() => (isLast ? value.trim() && send() : value.trim() && go(idx + 1))}
                  style={styles.input}
                />
                <MicButton onText={(txt) => setDraft(session.id, item.id, value ? `${value} ${txt}` : txt)} />
              </View>
              <View style={{ flexDirection: 'row', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                <Text size={13} color={colors.muted} style={{ flex: 1, minWidth: 160 }}>
                  {t('checkpoint.typeOrSpeak')}
                </Text>
                {idx > 0 ? <PillButton label={t('checkpoint.back')} variant="secondary" size="sm" disabled={submit.isPending} onPress={() => go(idx - 1)} /> : null}
                {isLast ? (
                  <PillButton
                    label={submit.isPending ? t('checkpoint.scoring') : t('checkpoint.submit')}
                    size="sm"
                    loading={submit.isPending}
                    disabled={items.some((it) => !(drafts[it.id] ?? '').trim())}
                    onPress={send}
                  />
                ) : (
                  <PillButton label={t('checkpoint.nextQuestion')} size="sm" disabled={!value.trim()} onPress={() => go(idx + 1)} />
                )}
              </View>
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
