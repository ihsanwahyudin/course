import { useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { Headline } from '@/components/Headline';
import { ErrorState, LoadingState, pad2, ProgressBar, SectionHeader } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { GradientText } from '@/components/ui/GradientText';
import { MeshBlob } from '@/components/ui/Decor';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { phaseLabel } from '@/lib/api';
import { useProgress } from '@/lib/queries';
import { colors, fonts, radius } from '@/theme/tokens';

export default function ProgressScreen() {
  const router = useRouter();
  const { data, isPending, isError, error, refetch, isRefetching } = useProgress();
  const { wide, gap, contentWidth } = useLayout();
  const left = wide ? span(contentWidth, gap, 3, 1) : contentWidth;
  const right = wide ? span(contentWidth, gap, 3, 2) : contentWidth;

  return (
    <Screen onRefresh={refetch} refreshing={isRefetching}>
      <SectionHeader
        wide={wide}
        label="Progress"
        title={<Headline size={wide ? 60 : 36} before="What you've" gradient="mastered" after=" so far." />}
        description="Mastery comes from assessments scored against each lesson's objectives — not from streaks or time spent."
      />
      {isPending ? (
        <LoadingState label="Loading progress" />
      ) : isError ? (
        <ErrorState title="Can't load progress" message={error.message} onRetry={refetch} />
      ) : (
        <View style={{ flexDirection: wide ? 'row' : 'column', gap, alignItems: 'flex-start' }}>
          <View style={{ width: left, gap }}>
            <Card tone="ink" reveal={0} padding={wide ? 32 : 24} style={{ gap: 28 }} decoration={<MeshBlob color="rgba(91,76,245,0.5)" size={240} style={{ right: -70, top: -70 }} />}>
              <Eyebrow color="rgba(255,255,255,0.6)">Skills mastered</Eyebrow>
              <View>
                <View style={{ flexDirection: 'row', alignItems: 'flex-end' }}>
                  <Heading size={72} color={colors.white} tabular>
                    {pad2(data.mastered.length)}
                  </Heading>
                  <GradientText animated={false} style={{ fontFamily: fonts.semibold, fontSize: 32, lineHeight: 42, letterSpacing: -1.4 }}>
                    {`/${data.summary.targetNodes}`}
                  </GradientText>
                </View>
                <Text color="rgba(255,255,255,0.7)" style={{ marginTop: 8 }}>
                  on the way to {data.learner.targetLevel} · now at {data.learner.currentLevel}
                </Text>
              </View>
              <View style={{ borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.1)' }}>
                {data.mastered.length === 0 ? (
                  <Text size={14} color="rgba(255,255,255,0.6)" style={{ paddingTop: 14 }}>
                    Nothing yet — your first mastered skill will appear here.
                  </Text>
                ) : (
                  data.mastered.map((m) => (
                    <View key={m.nodeId} style={styles.inkRow}>
                      <View style={{ flex: 1 }}>
                        <Text size={14} weight="medium" color={colors.white}>
                          {m.title}
                        </Text>
                        <Text size={12} color="rgba(255,255,255,0.5)">
                          {m.level} · {new Date(m.masteredAt).toLocaleDateString()}
                        </Text>
                      </View>
                      <Text size={12} color="rgba(255,255,255,0.5)" style={{ fontFamily: fonts.mono }}>
                        {m.score ?? '—'}
                      </Text>
                    </View>
                  ))
                )}
              </View>
            </Card>
            <Card tone="lavender" reveal={1} style={{ gap: 16 }}>
              <Eyebrow color="rgba(17,17,19,0.6)">Last 7 days</Eyebrow>
              <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
                {[
                  ['Replies', data.weekly.messages],
                  ['Attempts', data.weekly.attempts],
                  ['Mastered', data.weekly.mastered],
                ].map(([k, v]) => (
                  <View key={k}>
                    <Heading size={40} tabular>
                      {pad2(v as number)}
                    </Heading>
                    <Text size={13} color="rgba(17,17,19,0.7)" style={{ marginTop: 4 }}>
                      {k}
                    </Text>
                  </View>
                ))}
              </View>
            </Card>
          </View>

          <View style={{ width: right, gap }}>
            {data.openSessions.length > 0 ? (
              <Card reveal={1} style={{ gap: 14 }}>
                <Eyebrow>Pick up where you left off</Eyebrow>
                {data.openSessions.map((o, i) => (
                  <View key={o.sessionId} style={[styles.row, i > 0 && styles.divider]}>
                    <View style={{ flex: 1 }}>
                      <Text weight="medium" tight>
                        {o.title}
                      </Text>
                      <Eyebrow>{phaseLabel[o.phase]}</Eyebrow>
                    </View>
                    <PillButton label="Continue" size="sm" icon="arrowUpRight" onPress={() => router.push(`/learn/${o.nodeId}`)} />
                  </View>
                ))}
              </Card>
            ) : null}

            <Card reveal={2} padding={wide ? 32 : 24} style={{ gap: 16 }}>
              <View style={[styles.row, { justifyContent: 'space-between' }]}>
                <Eyebrow>Assessment history</Eyebrow>
                <Eyebrow>{pad2(data.recentAttempts.length)} recent</Eyebrow>
              </View>
              {data.recentAttempts.length === 0 ? (
                <Text color={colors.muted}>No assessments yet. Each lesson ends with a short assessment; your scores and feedback will be listed here.</Text>
              ) : (
                data.recentAttempts.map((a, i) => (
                  <View key={a.id} style={[{ gap: 10 }, i > 0 && styles.divider]}>
                    <View style={styles.row}>
                      <View style={[styles.score, { backgroundColor: a.passed ? colors.mint : colors.peach }]}>
                        <Text weight="semibold" tabular>
                          {a.score}
                        </Text>
                      </View>
                      <View style={{ flex: 1 }}>
                        <Text weight="medium" tight>
                          {a.title}
                        </Text>
                        <Eyebrow>
                          {a.passed ? 'Passed' : 'Below ' + a.threshold} · {new Date(a.createdAt).toLocaleString()}
                        </Eyebrow>
                      </View>
                    </View>
                    <Text size={14} color={colors.muted}>
                      {a.feedback}
                    </Text>
                    <View style={{ gap: 6 }}>
                      {a.objectives.map((o) => {
                        const v = a.objectiveScores[o.id] ?? 0;
                        return (
                          <View key={o.id} style={{ gap: 4 }}>
                            <View style={[styles.row, { justifyContent: 'space-between' }]}>
                              <Text size={12} color={colors.muted} style={{ flex: 1 }} numberOfLines={1}>
                                {o.text}
                              </Text>
                              <Text size={12} tabular style={{ fontFamily: fonts.mono }}>
                                {v}
                              </Text>
                            </View>
                            <ProgressBar value={v} color={v >= a.threshold ? colors.emerald : colors.orange} height={4} />
                          </View>
                        );
                      })}
                    </View>
                  </View>
                ))
              )}
            </Card>

            <Card tone="sky" reveal={3} style={{ gap: 14 }}>
              <Eyebrow color="rgba(17,17,19,0.6)">By level</Eyebrow>
              {data.summary.levels.map((l) => (
                <View key={l.level} style={[styles.row, { gap: 12 }]}>
                  <View style={styles.levelPill}>
                    <Text size={12} weight="semibold" color={colors.white}>
                      {l.level}
                    </Text>
                  </View>
                  <View style={{ flex: 1 }}>
                    <ProgressBar value={l.total ? (l.mastered / l.total) * 100 : 0} color={colors.accent} track="rgba(17,17,19,0.08)" />
                  </View>
                  <Text size={12} tabular color="rgba(17,17,19,0.7)">
                    {l.mastered}/{l.total} · {l.ready} ready
                  </Text>
                </View>
              ))}
            </Card>
          </View>
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  divider: { borderTopWidth: 1, borderTopColor: colors.line, paddingTop: 14 },
  inkRow: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.1)' },
  score: { width: 44, height: 44, borderRadius: radius.pill, alignItems: 'center', justifyContent: 'center' },
  levelPill: { backgroundColor: colors.ink, borderRadius: radius.pill, paddingHorizontal: 10, paddingVertical: 4, minWidth: 40, alignItems: 'center' },
});
