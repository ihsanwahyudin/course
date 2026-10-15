import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { Icon, type IconName } from '@/components/ui/Icon';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Attempt, AttemptSummary, CheckpointReview, ItemReview } from '@/lib/api';
import { useAttempt } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';
import { LinkButton } from './StepParts';
import { RichText } from './RichText';

const look: Record<ItemReview['verdict'], { icon: IconName; fg: string; bg: string }> = {
  correct: { icon: 'check', fg: colors.emeraldDeep, bg: colors.mint },
  partial: { icon: 'undo', fg: colors.amber, bg: colors.amberSoft },
  wrong: { icon: 'close', fg: colors.danger, bg: colors.peach },
};

function Item({ it, n }: { it: ItemReview; n: number }) {
  const t = useT();
  const l = look[it.verdict];
  return (
    <View style={styles.item}>
      <View style={{ flexDirection: 'row', alignItems: 'flex-start', gap: 10 }}>
        <View style={[styles.badge, { backgroundColor: l.bg }]}>
          <Icon name={l.icon} size={13} color={l.fg} strokeWidth={2.5} />
        </View>
        <View style={{ flex: 1, gap: 6 }}>
          <Text size={15} weight="medium" tight>
            {n}. {it.prompt}
          </Text>
          {it.listen ? (
            <Text size={13} color={colors.muted}>
              {t('correction.heard', { text: it.listen })}
            </Text>
          ) : null}
          <Text size={14}>
            <Text size={14} color={colors.muted}>
              {t('correction.yourAnswer')}{' '}
            </Text>
            <Text size={14} weight="medium" color={l.fg}>
              {it.answer}
            </Text>
          </Text>
          {it.verdict !== 'correct' ? (
            <Text size={14}>
              <Text size={14} color={colors.muted}>
                {t('correction.correctAnswer')}{' '}
              </Text>
              <Text size={14} weight="medium" color={colors.emeraldDeep}>
                {it.correctAnswer}
              </Text>
            </Text>
          ) : null}
          <RichText size={14} lineHeight={1.5}>
            {it.explanation}
          </RichText>
          {it.keyFixed ? (
            <Text size={13} color={colors.muted}>
              {t('correction.keyFixed')}
            </Text>
          ) : null}
        </View>
        <Text size={13} color={l.fg} tabular>
          {t(`correction.${it.verdict}`)}
        </Text>
      </View>
    </View>
  );
}

// Every answer of an attempt with its correction and explanation, plus
// what the learner does well and what to practise.
export function ReviewList({ review, feedback }: { review: CheckpointReview; feedback?: string }) {
  const t = useT();
  return (
    <View style={{ gap: 10 }}>
      {feedback ? <Text size={15} style={{ lineHeight: 23 }}>{feedback}</Text> : null}
      {review.strengths.length || review.toReview.length ? (
        <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 16 }}>
          {review.strengths.length ? (
            <View style={{ flex: 1, minWidth: 200, gap: 6 }}>
              <Eyebrow color={colors.emeraldDeep}>{t('correction.strengths')}</Eyebrow>
              {review.strengths.map((x, i) => (
                <Text key={i} size={14}>
                  · {x}
                </Text>
              ))}
            </View>
          ) : null}
          {review.toReview.length ? (
            <View style={{ flex: 1, minWidth: 200, gap: 6 }}>
              <Eyebrow color={colors.danger}>{t('correction.toReview')}</Eyebrow>
              {review.toReview.map((x, i) => (
                <Text key={i} size={14}>
                  · {x}
                </Text>
              ))}
            </View>
          ) : null}
        </View>
      ) : null}
      {review.items.map((it, i) => (
        <Item key={it.id} it={it} n={i + 1} />
      ))}
    </View>
  );
}

// An attempt's corrections behind a "see the corrections" toggle.
export function AttemptReview({ attempt, open: initiallyOpen = false }: { attempt: Attempt; open?: boolean }) {
  const t = useT();
  const [open, setOpen] = useState(initiallyOpen);
  if (!attempt.review) return null;
  return (
    <View style={{ gap: 12 }}>
      <LinkButton
        label={open ? t('correction.hide') : t('correction.show', { n: attempt.review.items.length })}
        icon={open ? 'arrowUp' : 'list'}
        onPress={() => setOpen(!open)}
      />
      {open ? <ReviewList review={attempt.review} feedback={attempt.feedback} /> : null}
    </View>
  );
}

// The lesson's checkpoint history, oldest first; tap one to see its corrections.
export function AttemptHistory({ attempts, current }: { attempts: AttemptSummary[]; current?: string }) {
  const t = useT();
  const [picked, setPicked] = useState<string | null>(null);
  const detail = useAttempt(picked);
  if (attempts.length < 2) return null;
  const ordered = [...attempts].reverse();
  return (
    <View style={styles.history}>
      <Eyebrow>{t('history.title')}</Eyebrow>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
        {ordered.map((a, i) => {
          const on = picked === a.id || (!picked && a.id === current);
          return (
            <Pressable
              key={a.id}
              onPress={() => setPicked(picked === a.id ? null : a.id)}
              accessibilityRole="button"
              accessibilityState={{ selected: on }}
              style={[styles.chip, on && { borderColor: colors.accent, backgroundColor: colors.accentSoft }]}
            >
              <Text size={13} color={on ? colors.accent : colors.muted}>
                {t('history.entry', { n: i + 1 })}
              </Text>
              <Text size={14} weight="medium" color={a.passed ? colors.emeraldDeep : colors.danger} tabular>
                {a.score}
              </Text>
              {a.practice ? (
                <Text size={12} color={colors.muted}>
                  · {t('checkpoint.practiceTag')}
                </Text>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      {picked && detail.data?.review ? <ReviewList review={detail.data.review} feedback={detail.data.feedback} /> : null}
      {picked && detail.isPending ? (
        <Text size={13} color={colors.muted}>
          {t('common.loading')}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  item: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: radius.xxl, padding: 14 },
  badge: { width: 24, height: 24, borderRadius: 12, alignItems: 'center', justifyContent: 'center', marginTop: 1 },
  history: { gap: 10 },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 6, borderWidth: 1, borderColor: colors.line, backgroundColor: colors.card, borderRadius: radius.pill, paddingHorizontal: 12, paddingVertical: 7 },
});
