import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { PageHeader } from '@/components/PageHeader';
import { ErrorState, LoadingState, ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT, type TKey } from '@/i18n';
import type { ReviewGrade } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useDueFlashcards, useReviewFlashcard } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';

const grades: { grade: ReviewGrade; label: TKey; hint: TKey; bg: string }[] = [
  { grade: 'again', label: 'review.again', hint: 'review.againHint', bg: colors.peach },
  { grade: 'good', label: 'review.good', hint: 'review.goodHint', bg: colors.accentSoft },
  { grade: 'easy', label: 'review.easy', hint: 'review.easyHint', bg: colors.mint },
];

// Spaced-repetition review of saved flashcards: show the front, reveal the
// back, grade it, and the server schedules the next time.
export default function ReviewScreen() {
  const router = useRouter();
  const t = useT();
  const errorText = useErrorText();
  const { wide } = useLayout();
  const { data, isPending, isError, error, refetch } = useDueFlashcards();
  const review = useReviewFlashcard();
  const [index, setIndex] = useState(0);
  const [shown, setShown] = useState(false);

  const cards = data?.cards ?? [];
  const card = cards[index];
  const grade = (g: ReviewGrade) =>
    review.mutate(
      { id: card.id, grade: g },
      {
        onSuccess: () => {
          setShown(false);
          setIndex((i) => i + 1);
        },
      },
    );

  return (
    <Screen maxWidth={760}>
      <PageHeader label={t('review.title')} title={wide ? t('review.titleWide') : t('review.title')} description={t('review.description')} />
      {isPending ? (
        <LoadingState />
      ) : isError ? (
        <ErrorState message={errorText(error)} onRetry={refetch} />
      ) : cards.length === 0 || !card ? (
        <Card style={{ gap: 12 }}>
          <Heading size={26}>{cards.length === 0 ? t('review.empty') : t('review.done')}</Heading>
          <Text color={colors.muted}>{cards.length === 0 ? t('review.emptyText') : t('review.doneText')}</Text>
          <View style={{ flexDirection: 'row', gap: 8, flexWrap: 'wrap' }}>
            {cards.length > 0 ? <PillButton label={t('common.tryAgain')} variant="secondary" onPress={() => { setIndex(0); refetch(); }} /> : null}
            <PillButton label={t('common.browseCourses')} icon="arrowUpRight" onPress={() => router.push('/courses')} />
          </View>
        </Card>
      ) : (
        <View style={{ gap: 14 }}>
          <View style={{ gap: 8 }}>
            <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
              <Eyebrow>{t('review.from', { lesson: card.nodeTitle, course: card.courseTitle })}</Eyebrow>
              <Eyebrow>{t('review.left', { n: cards.length - index })}</Eyebrow>
            </View>
            <ProgressBar value={(index * 100) / cards.length} />
          </View>
          <Pressable onPress={() => setShown(true)} disabled={shown} accessibilityRole="button" accessibilityLabel={shown ? card.back : `${card.front}. ${t('review.showAnswer')}`}>
            <Card padding={28} spotlight={false} style={{ minHeight: 260, alignItems: 'center', justifyContent: 'center', gap: 18 }}>
              <Text size={24} weight="medium" tight style={{ textAlign: 'center' }}>
                {card.front}
              </Text>
              {shown ? (
                <>
                  <View style={styles.rule} />
                  <Text size={22} color={colors.accent} weight="medium" tight style={{ textAlign: 'center' }} accessibilityLiveRegion="polite">
                    {card.back}
                  </Text>
                </>
              ) : (
                <Eyebrow>{t('review.showAnswer')}</Eyebrow>
              )}
            </Card>
          </Pressable>
          {shown ? (
            <View style={{ flexDirection: 'row', gap: 10 }}>
              {grades.map((g) => (
                <Pressable
                  key={g.grade}
                  onPress={() => grade(g.grade)}
                  disabled={review.isPending}
                  accessibilityRole="button"
                  style={[styles.grade, { backgroundColor: g.bg, opacity: review.isPending ? 0.6 : 1 }]}
                >
                  <Text weight="medium">{t(g.label)}</Text>
                  <Text size={12} color={colors.muted}>
                    {t(g.hint)}
                  </Text>
                </Pressable>
              ))}
            </View>
          ) : (
            <PillButton label={t('review.showAnswer')} onPress={() => setShown(true)} />
          )}
          {review.isError ? (
            <Text size={14} color={colors.danger}>
              {errorText(review.error)}
            </Text>
          ) : null}
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  rule: { width: 48, height: 2, borderRadius: 1, backgroundColor: colors.line },
  grade: { flex: 1, alignItems: 'center', gap: 2, paddingVertical: 14, borderRadius: radius.xxl + 4 },
});
