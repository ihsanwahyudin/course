import { useRouter } from 'expo-router';
import { useState } from 'react';
import { StyleSheet, View } from 'react-native';

import { Headline } from '@/components/Headline';
import { PageHeader } from '@/components/PageHeader';
import { Chip, ErrorState, LoadingState, ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useFormatDate, useT } from '@/i18n';
import { useErrorText } from '@/lib/errors';
import { useCourseProgress, useDashboard } from '@/lib/queries';
import { colors, fonts, radius } from '@/theme/tokens';

export default function ProgressScreen() {
  const router = useRouter();
  const dashboard = useDashboard();
  const enrolled = dashboard.data?.courses ?? [];
  const [picked, setPicked] = useState<string>();
  const course = picked ?? enrolled[0]?.slug;
  const { data, isPending: progressPending, isError, error, refetch, isRefetching } = useCourseProgress(course);
  const isPending = dashboard.isPending || (!!course && progressPending);
  const weekly = dashboard.data?.weekly ?? { steps: 0, attempts: 0, mastered: 0 };
  const { wide, gap, contentWidth } = useLayout();
  const t = useT();
  const errorText = useErrorText();
  const formatDate = useFormatDate();
  const left = wide ? span(contentWidth, gap, 3, 1) : contentWidth;
  const right = wide ? span(contentWidth, gap, 3, 2) : contentWidth;

  return (
    <Screen onRefresh={refetch} refreshing={isRefetching}>
      <PageHeader
        label={t('progress.title')}
        title={wide ? <Headline size={60} before={t('progress.before')} highlight={t('progress.gradient')} after={t('progress.after')} /> : t('progress.title')}
        description={t('progress.description')}
      />
      {enrolled.length > 1 ? (
        <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginBottom: 16 }}>
          {enrolled.map((c) => (
            <Chip key={c.slug} label={c.title} active={c.slug === course} onPress={() => setPicked(c.slug)} />
          ))}
        </View>
      ) : null}
      {isPending ? (
        <LoadingState label={t('progress.loading')} />
      ) : dashboard.isError ? (
        <ErrorState title={t('progress.loadError')} message={errorText(dashboard.error)} onRetry={dashboard.refetch} />
      ) : !course || !data ? (
        <Card style={{ gap: 12 }}>
          <Eyebrow color="rgba(17,17,19,0.6)">{t('progress.nothingYet')}</Eyebrow>
          <Heading size={28}>{t('progress.startToTrack')}</Heading>
          <Text color="rgba(17,17,19,0.7)">{t('progress.startToTrackText')}</Text>
          <PillButton label={t('common.browseCourses')} icon="arrowUpRight" onPress={() => router.push('/courses')} />
        </Card>
      ) : isError ? (
        <ErrorState title={t('progress.loadError')} message={errorText(error)} onRetry={refetch} />
      ) : (
        <View style={{ flexDirection: wide ? 'row' : 'column', gap, alignItems: 'flex-start' }}>
          <View style={{ width: left, gap }}>
            <Card tone="ink" reveal={0} padding={wide ? 32 : 24} style={{ gap: 28 }}>
              <Eyebrow color="rgba(255,255,255,0.6)">{t('progress.skillsMastered')}</Eyebrow>
              <View>
                <View style={{ flexDirection: 'row', alignItems: 'flex-end' }}>
                  <Heading size={72} color={colors.white} tabular>
                    {data.mastered.length}
                  </Heading>
                  <Heading size={32} color="rgba(255,255,255,0.45)" style={{ marginBottom: 6 }}>
                    {`/${data.summary.targetNodes}`}
                  </Heading>
                </View>
                <Text color="rgba(255,255,255,0.7)" style={{ marginTop: 8 }}>
                  {t('progress.onTheWay', { course: data.course.title, target: data.course.progress?.targetLevel ?? '', level: data.course.progress?.currentLevel ?? '' })}
                </Text>
              </View>
              <View style={{ borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.1)' }}>
                {data.mastered.length === 0 ? (
                  <Text size={14} color="rgba(255,255,255,0.6)" style={{ paddingTop: 14 }}>
                    {t('progress.noneMastered')}
                  </Text>
                ) : (
                  data.mastered.map((m) => (
                    <View key={m.nodeId} style={styles.inkRow}>
                      <View style={{ flex: 1 }}>
                        <Text size={14} weight="medium" color={colors.white}>
                          {m.title}
                        </Text>
                        <Text size={12} color="rgba(255,255,255,0.5)">
                          {m.level} · {formatDate(m.masteredAt)}
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
            <Card reveal={1} style={{ gap: 16 }}>
              <Eyebrow color="rgba(17,17,19,0.6)">{t('progress.last7')}</Eyebrow>
              <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
                {[
                  [t('progress.steps'), weekly.steps],
                  [t('progress.attempts'), weekly.attempts],
                  [t('progress.masteredShort'), weekly.mastered],
                ].map(([k, v]) => (
                  <View key={k}>
                    <Heading size={40} tabular>
                      {v}
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
                <Eyebrow>{t('dashboard.pickUp')}</Eyebrow>
                {data.openSessions.map((o, i) => (
                  <View key={o.sessionId} style={[styles.row, i > 0 && styles.divider]}>
                    <View style={{ flex: 1 }}>
                      <Text weight="medium" tight>
                        {o.title}
                      </Text>
                      <Eyebrow>{t(`phase.${o.phase}`)}</Eyebrow>
                    </View>
                    <PillButton label={t('common.continue')} size="sm" icon="arrowUpRight" onPress={() => router.push(`/learn/${o.courseSlug}/${o.nodeId}`)} />
                  </View>
                ))}
              </Card>
            ) : null}

            <Card reveal={2} padding={wide ? 32 : 24} style={{ gap: 16 }}>
              <View style={[styles.row, { justifyContent: 'space-between' }]}>
                <Eyebrow>{t('progress.history')}</Eyebrow>
                <Eyebrow>{t('common.recent', { n: data.recentAttempts.length })}</Eyebrow>
              </View>
              {data.recentAttempts.length === 0 ? (
                <Text color={colors.muted}>{t('progress.noHistory')}</Text>
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
                          {a.passed ? t('progress.passed') : t('progress.below', { n: a.threshold })} · {formatDate(a.createdAt, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}
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

            <Card reveal={3} style={{ gap: 14 }}>
              <Eyebrow>{t('progress.byLevel')}</Eyebrow>
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
                    {t('progress.levelLine', { done: l.mastered, total: l.total, ready: l.ready })}
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
