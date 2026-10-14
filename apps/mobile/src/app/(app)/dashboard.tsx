import { useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { ComingSoonCard, CourseCard } from '@/components/CourseCard';
import { Headline } from '@/components/Headline';
import { PageHeader } from '@/components/PageHeader';
import { Avatar } from '@/components/ui/FloatingNav';
import { ErrorState, LoadingState, ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { ArrowBadge, PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Span, Text } from '@/components/ui/Text';
import { dateLocale, useFormatDate, useLocale, useT } from '@/i18n';
import type { Dashboard } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useDashboard } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';


// 4 weeks × 7 days of activity, darker = more lesson steps/attempts that day.
function ActivityGrid({ days, cell }: { days: Dashboard['activity']; cell: number }) {
  const t = useT();
  const max = Math.max(1, ...days.map((d) => d.count));
  const weeks = [0, 1, 2, 3].map((w) => days.slice(w * 7, w * 7 + 7));
  return (
    <View style={{ gap: 5 }} accessibilityLabel={t('dashboard.activityLabel', { n: days.filter((d) => d.count > 0).length, total: days.length })}>
      {weeks.map((week, i) => (
        <View key={i} style={{ flexDirection: 'row', gap: 5 }}>
          {week.map((d) => {
            const level = d.count === 0 ? 0 : Math.ceil((d.count / max) * 3);
            const bg = ['rgba(17,17,19,0.07)', 'rgba(91,76,245,0.3)', 'rgba(91,76,245,0.6)', colors.accent][level];
            return <View key={d.date} style={{ width: cell, height: cell, borderRadius: 4, backgroundColor: bg }} />;
          })}
        </View>
      ))}
    </View>
  );
}

export default function DashboardScreen() {
  const router = useRouter();
  const { data, isPending, isError, error, refetch, isRefetching } = useDashboard();
  const { wide, gap, contentWidth, cols, width } = useLayout();
  const w = (n: number) => span(contentWidth, gap, cols, n);
  const t = useT();
  const errorText = useErrorText();
  const formatDate = useFormatDate();
  const locale = useLocale((s) => s.locale);
  const today = new Date().toLocaleDateString(dateLocale(locale), { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });

  if (isPending) {
    return (
      <Screen>
        <LoadingState label={t('dashboard.loading')} />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState title={t('dashboard.loadError')} message={errorText(error)} onRetry={refetch} />
      </Screen>
    );
  }

  const first = data.user.displayName.split(' ')[0];
  const open = data.openSessions[0];
  const nextCourse = data.courses.find((c) => c.progress?.nextNode);
  const continueTo = open
    ? { label: t('dashboard.continueTitle', { title: open.title }), go: () => router.push(`/learn/${open.courseSlug}/${open.nodeId}`) }
    : nextCourse
      ? { label: t('dashboard.startTitle', { title: nextCourse.progress!.nextNode!.title }), go: () => router.push(`/learn/${nextCourse.slug}/${nextCourse.progress!.nextNode!.id}`) }
      : null;
  const rowH = wide ? 248 : undefined;
  const h = (v: number | undefined) => (wide ? { height: v } : { minHeight: v });

  const hero = (
    <Card
      reveal={0}
      padding={wide ? 32 : 24}
      style={{ width: w(2), ...(wide ? { height: rowH! * 2 + gap } : {}), justifyContent: 'space-between', gap: 28 }}
    >
      {wide ? (
        // Phones already show name and date in the app header above.
        <View style={styles.row}>
          <Avatar name={data.user.displayName} size={48} />
          <View>
            <Text weight="medium" tight>
              {data.user.displayName}
            </Text>
            <Eyebrow>{today}</Eyebrow>
          </View>
        </View>
      ) : null}
      <View>
        <Headline
          size={wide ? (width >= 1024 ? 56 : 48) : 34}
          before={
            data.courses.length
              ? wide
                ? t('dashboard.keepBeforeWide', { name: first })
                : t('dashboard.keepBefore')
              : wide
                ? t('dashboard.pickBeforeWide', { name: first })
                : t('dashboard.pickBefore')
          }
          highlight={data.courses.length ? t('dashboard.keepGradient') : t('dashboard.pickGradient')}
          after={data.courses.length ? t('dashboard.keepAfter') : t('dashboard.pickAfter')}
        />
        <Text color={colors.muted} size={wide ? 17 : 15} style={{ marginTop: 16, maxWidth: 440 }}>
          {continueTo ? t('dashboard.readyText') : t('dashboard.emptyText')}
        </Text>
        <View style={[styles.row, { marginTop: 24, flexWrap: 'wrap', gap: 8 }]}>
          {continueTo ? <PillButton label={continueTo.label} icon="arrowUpRight" onPress={continueTo.go} /> : null}
          <PillButton label={t('common.browseCourses')} variant={continueTo ? 'secondary' : 'primary'} icon={continueTo ? undefined : 'arrowUpRight'} onPress={() => router.push('/courses')} />
        </View>
      </View>
    </Card>
  );

  const streakTile = (
    <Card reveal={1} style={{ width: w(1), ...h(rowH ?? 168), justifyContent: 'space-between', gap: 12 }}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow color="rgba(17,17,19,0.6)">{t('dashboard.streak')}</Eyebrow>
        <Icon name="flame" size={18} color={data.streak.current ? colors.orange : 'rgba(17,17,19,0.35)'} />
      </View>
      <Heading size={wide ? 72 : 52} tabular>
        {data.streak.current}
        <Span color={colors.accent}>{t('dashboard.dayUnit')}</Span>
      </Heading>
      <Text size={13} color="rgba(17,17,19,0.7)">
        {data.streak.activeToday ? t('dashboard.streakToday') : data.streak.current ? t('dashboard.streakKeep') : t('dashboard.streakStart')}
      </Text>
    </Card>
  );

  const masteredTile = (
    <Card reveal={3} style={{ width: w(1), ...h(rowH ?? 168), justifyContent: 'space-between', gap: 12 }}>
      <Eyebrow>{t('dashboard.mastered')}</Eyebrow>
      <View>
        <Heading size={wide ? 64 : 48} tabular>
          {data.totalMastered}
        </Heading>
        <Text size={13} color="rgba(17,17,19,0.6)" style={{ marginTop: 6 }}>
          {t('dashboard.masteredSub', { week: data.weekly.mastered, courses: data.courses.length })}
        </Text>
      </View>
    </Card>
  );

  const activityTile = (
    <Card
      reveal={2}
      style={{ width: w(wide ? 1 : 2), ...(wide ? { height: rowH! * 2 + gap } : {}), justifyContent: 'space-between', gap: 20 }}
    >
      <Eyebrow>{t('dashboard.last4Weeks')}</Eyebrow>
      <View style={{ alignItems: wide ? 'center' : 'flex-start' }}>
        <ActivityGrid days={data.activity} cell={wide ? 22 : 30} />
      </View>
      <View style={{ gap: 8 }}>
        {[
          [t('dashboard.steps7d'), data.weekly.steps],
          [t('dashboard.assessments7d'), data.weekly.attempts],
        ].map(([k, v]) => (
          <View key={k as string} style={[styles.row, { justifyContent: 'space-between' }]}>
            <Text size={13} color={colors.muted}>
              {k}
            </Text>
            <Text size={18} weight="semibold" tight tabular>
              {v}
            </Text>
          </View>
        ))}
      </View>
    </Card>
  );

  const coursesW = data.courses.length > 1 || !wide ? w(2) : w(2);
  const coursesBlock = data.courses.length ? (
    data.courses.map((c, i) => <CourseCard key={c.slug} course={c} reveal={4 + i} style={{ width: coursesW }} />)
  ) : (
    <Card reveal={4} style={{ width: w(2), gap: 12 }}>
      <Eyebrow color="rgba(17,17,19,0.6)">{t('dashboard.yourCourses')}</Eyebrow>
      <Heading size={28}>{t('dashboard.noCourses')}</Heading>
      <Text color="rgba(17,17,19,0.7)">{t('dashboard.noCoursesText')}</Text>
      <PillButton label={t('common.browseCourses')} icon="arrowUpRight" onPress={() => router.push('/courses')} />
    </Card>
  );

  const openTile = (
    <Card reveal={5} style={{ width: w(2), gap: 14 }}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>{t('dashboard.pickUp')}</Eyebrow>
        <Eyebrow>{t('common.open', { n: data.openSessions.length })}</Eyebrow>
      </View>
      {data.openSessions.length === 0 ? (
        <Text color={colors.muted}>{t('dashboard.noOpen')}</Text>
      ) : (
        data.openSessions.slice(0, 4).map((o, i) => (
          <View key={o.sessionId} style={[styles.row, i > 0 && styles.divider]}>
            <View style={styles.phaseBadge}>
              <Icon name="play" size={14} color={colors.accent} />
            </View>
            <View style={{ flex: 1 }}>
              <Text weight="medium" tight numberOfLines={1}>
                {o.title}
              </Text>
              <Eyebrow>
                {o.courseTitle} · {t(`phase.${o.phase}`)}
              </Eyebrow>
            </View>
            <PillButton label={t('common.resume')} size="sm" variant="secondary" onPress={() => router.push(`/learn/${o.courseSlug}/${o.nodeId}`)} />
          </View>
        ))
      )}
    </Card>
  );

  const attemptsTile = (
    <Card reveal={6} style={{ width: w(2), gap: 14 }} padding={wide ? 28 : 20}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>{t('dashboard.recentAssessments')}</Eyebrow>
        <PillButton label={t('dashboard.allProgress')} size="sm" variant="secondary" onPress={() => router.push('/progress')} />
      </View>
      {data.recentAttempts.length === 0 ? (
        <Text color={colors.muted}>{t('dashboard.noAttempts')}</Text>
      ) : (
        data.recentAttempts.slice(0, 4).map((a, i) => (
          <View key={a.id} style={[styles.row, i > 0 && styles.divider]}>
            <View style={[styles.score, { backgroundColor: a.passed ? colors.mint : colors.peach }]}>
              <Text size={14} weight="semibold" tabular>
                {a.score}
              </Text>
            </View>
            <View style={{ flex: 1, gap: 4 }}>
              <Text weight="medium" tight numberOfLines={1}>
                {a.title}
              </Text>
              <ProgressBar value={a.score} color={a.passed ? colors.emerald : colors.orange} height={3} />
              <Eyebrow>
                {a.courseTitle} · {a.passed ? t('status.mastered') : t('dashboard.reviewNeeded')} · {formatDate(a.createdAt)}
              </Eyebrow>
            </View>
          </View>
        ))
      )}
    </Card>
  );

  const flashcardTile = (
    <Card
      reveal={7}
      onPress={() => router.push('/review')}
      accessibilityLabel={t('review.reviewNow')}
      style={{ width: w(wide ? 1 : 2), minHeight: 180, justifyContent: 'space-between', gap: 12 }}
    >
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>{t('review.tile')}</Eyebrow>
        <Icon name="book" size={18} color={data.flashcards.due ? colors.accent : colors.muted} />
      </View>
      <View style={{ gap: 4 }}>
        <Heading size={wide ? 44 : 40} tabular>
          {data.flashcards.due}
        </Heading>
        <Text size={13} color={colors.muted}>
          {data.flashcards.due ? t('review.dueToday', { n: data.flashcards.due }) : t('review.noneDue')} · {t('review.total', { n: data.flashcards.total })}
        </Text>
      </View>
      {data.flashcards.due ? <PillButton label={t('review.reviewNow')} size="sm" icon="arrowUpRight" onPress={() => router.push('/review')} style={{ alignSelf: 'flex-start' }} /> : null}
    </Card>
  );

  const exploreTile =
    data.explore.length > 0 ? (
      <CourseCard course={data.explore[0]} reveal={7} style={{ width: w(wide ? 1 : 2) }} />
    ) : (
      <Card reveal={7} onPress={() => router.push('/courses')} accessibilityLabel={t('dashboard.exploreLabel')} style={{ width: w(wide ? 1 : 2), minHeight: 180, justifyContent: 'space-between' }}>
        <View style={[styles.row, { justifyContent: 'space-between' }]}>
          <Eyebrow color="rgba(17,17,19,0.6)">{t('dashboard.explore')}</Eyebrow>
          <ArrowBadge size={34} />
        </View>
        <View>
          <Heading size={28}>{t('dashboard.moreSoon')}</Heading>
          <Text size={13} color="rgba(17,17,19,0.7)" style={{ marginTop: 6 }}>
            {t('dashboard.moreSoonText')}
          </Text>
        </View>
      </Card>
    );

  return (
    <Screen onRefresh={refetch} refreshing={isRefetching}>
      {!wide ? <PageHeader label={t('nav.home')} eyebrow={today} title={t('dashboard.hi', { name: first })} aside={<Avatar name={data.user.displayName} size={40} />} /> : null}
      {wide ? (
        <View style={{ gap }}>
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {hero}
            <View style={{ gap }}>
              {streakTile}
              {masteredTile}
            </View>
            {activityTile}
          </View>
          <View style={[styles.row, { gap, alignItems: 'stretch', flexWrap: 'wrap' }]}>
            {coursesBlock}
            {openTile}
          </View>
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {attemptsTile}
            {flashcardTile}
            {exploreTile}
          </View>
        </View>
      ) : (
        <View style={{ gap }}>
          {hero}
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {streakTile}
            {masteredTile}
          </View>
          {coursesBlock}
          {openTile}
          {flashcardTile}
          {activityTile}
          {attemptsTile}
          {exploreTile}
          {data.explore.length === 0 ? null : <ComingSoonCard />}
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  divider: { borderTopWidth: 1, borderTopColor: colors.line, paddingTop: 12 },
  score: { width: 44, height: 44, borderRadius: radius.pill, alignItems: 'center', justifyContent: 'center' },
  phaseBadge: { width: 36, height: 36, borderRadius: radius.pill, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
});
