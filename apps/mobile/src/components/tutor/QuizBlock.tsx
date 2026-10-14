import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { QuizQuestion } from '@/lib/api';
import { colors, radius } from '@/theme/tokens';

// A practice quiz from the helper, checked on the device. It never counts
// towards mastery (only the final quiz does).
export function QuizBlock({ questions }: { questions: QuizQuestion[] }) {
  const t = useT();
  const [picked, setPicked] = useState<Record<number, number>>({});
  const answered = Object.keys(picked).length;
  const correct = questions.filter((q, i) => picked[i] === q.answer).length;
  return (
    <View style={styles.wrap}>
      <View style={{ flexDirection: 'row', justifyContent: 'space-between', gap: 8, flexWrap: 'wrap' }}>
        <Eyebrow color={colors.accent}>{t('helper.practiceOnly')}</Eyebrow>
        {answered === questions.length ? <Eyebrow color={colors.emeraldDeep}>{t('helper.score', { n: correct, total: questions.length })}</Eyebrow> : null}
      </View>
      {questions.map((q, i) => {
        const done = picked[i] !== undefined;
        return (
          <View key={i} style={{ gap: 8 }}>
            <Text size={12} color={colors.muted}>
              {t('helper.questionOf', { n: i + 1, total: questions.length })}
            </Text>
            <Text size={15} weight="medium" tight>
              {q.prompt}
            </Text>
            <View style={{ gap: 6 }} accessibilityRole="radiogroup">
              {q.options.map((o, j) => {
                const isAnswer = done && j === q.answer;
                const wrongPick = done && j === picked[i] && j !== q.answer;
                return (
                  <Pressable
                    key={j}
                    disabled={done}
                    onPress={() => setPicked((p) => ({ ...p, [i]: j }))}
                    accessibilityRole="radio"
                    accessibilityState={{ selected: picked[i] === j, disabled: done }}
                    style={[
                      styles.option,
                      isAnswer && { backgroundColor: colors.mint, borderColor: colors.emerald },
                      wrongPick && { backgroundColor: colors.peach, borderColor: colors.danger },
                    ]}
                  >
                    <Text size={14}>{o}</Text>
                  </Pressable>
                );
              })}
            </View>
            {done && q.explain ? (
              <Text size={13} color={colors.muted} accessibilityLiveRegion="polite">
                {q.explain}
              </Text>
            ) : null}
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 16, borderWidth: 1, borderColor: colors.line, borderRadius: 20, padding: 16, backgroundColor: colors.card },
  option: { borderWidth: 1.5, borderColor: colors.line, borderRadius: radius.xxl, paddingHorizontal: 14, paddingVertical: 10, backgroundColor: colors.card },
});
