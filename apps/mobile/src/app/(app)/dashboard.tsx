import { useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { ComingSoonCard, CourseCard } from '@/components/CourseCard';
import { Headline } from '@/components/Headline';
import { PageHeader } from '@/components/PageHeader';
import { Avatar } from '@/components/ui/FloatingNav';
import { ErrorState, LoadingState, pad2, ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern, LiveDot, MeshBlob } from '@/components/ui/Decor';
import { Icon } from '@/components/ui/Icon';
import { ArrowBadge, PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Span, Text } from '@/components/ui/Text';
import { phaseLabel, type Dashboard } from '@/lib/api';
import { useDashboard } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';

const today = () => new Date().toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });

// 4 weeks × 7 days of activity, darker = more replies/attempts that day.
function ActivityGrid({ days, cell }: { days: Dashboard['activity']; cell: number }) {
  const max = Math.max(1, ...days.map((d) => d.count));
  const weeks = [0, 1, 2, 3].map((w) => days.slice(w * 7, w * 7 + 7));
  return (
    <View style={{ gap: 5 }} accessibilityLabel={`Active on ${days.filter((d) => d.count > 0).length} of the last ${days.length} days`}>
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

  if (isPending) {
    return (
      <Screen>
        <LoadingState label="Loading your dashboard" />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState title="Can't load your dashboard" message={error.message} onRetry={refetch} />
      </Screen>
    );
  }

  const first = data.user.displayName.split(' ')[0];
  const open = data.openSessions[0];
  const nextCourse = data.courses.find((c) => c.progress?.nextNode);
  const continueTo = open
    ? { label: `Continue: ${open.title}`, go: () => router.push(`/learn/${open.courseSlug}/${open.nodeId}`) }
    : nextCourse
      ? { label: `Start: ${nextCourse.progress!.nextNode!.title}`, go: () => router.push(`/learn/${nextCourse.slug}/${nextCourse.progress!.nextNode!.id}`) }
      : null;
  const rowH = wide ? 248 : undefined;
  const h = (v: number | undefined) => (wide ? { height: v } : { minHeight: v });

  const hero = (
    <Card
      reveal={0}
      padding={wide ? 32 : 24}
      style={{ width: w(2), ...(wide ? { height: rowH! * 2 + gap } : {}), justifyContent: 'space-between', gap: 28 }}
      decoration={
        <>
          <MeshBlob color={colors.lavender} size={320} style={{ right: -110, top: -110 }} />
          <MeshBlob color={colors.peach} size={260} style={{ right: 20, bottom: -130 }} delay={2500} />
        </>
      }
    >
      {wide ? (
        // Phones already show name and date in the app header above.
        <View style={styles.row}>
          <Avatar name={data.user.displayName} size={48} />
          <View>
            <Text weight="medium" tight>
              {data.user.displayName}
            </Text>
            <Eyebrow>{today()}</Eyebrow>
          </View>
        </View>
      ) : null}
      <View>
        <Headline
          size={wide ? (width >= 1024 ? 56 : 48) : 34}
          before={data.courses.length ? (wide ? `Hi ${first}, keep` : 'Keep') : wide ? `Welcome, ${first}. Pick a` : 'Pick a'}
          gradient={data.courses.length ? 'your streak' : 'roadmap'}
          after={data.courses.length ? ' going.' : ' to begin.'}
        />
        <Text color={colors.muted} size={wide ? 17 : 15} style={{ marginTop: 16, maxWidth: 440 }}>
          {continueTo
            ? 'Your next step is ready. Lessons take a few minutes, and every skill you master unlocks the next ones.'
            : 'Choose a course from the catalog. Your lessons, scores and progress will show up here.'}
        </Text>
        <View style={[styles.row, { marginTop: 24, flexWrap: 'wrap', gap: 8 }]}>
          {continueTo ? <PillButton label={continueTo.label} icon="arrowUpRight" onPress={continueTo.go} /> : null}
          <PillButton label="Browse courses" variant={continueTo ? 'secondary' : 'primary'} icon={continueTo ? undefined : 'arrowUpRight'} onPress={() => router.push('/courses')} />
        </View>
      </View>
    </Card>
  );

  const streakTile = (
    <Card tone="lavender" reveal={1} style={{ width: w(1), ...h(rowH ?? 168), justifyContent: 'space-between', gap: 12 }}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow color="rgba(17,17,19,0.6)">Streak</Eyebrow>
        <Icon name="flame" size={18} color={data.streak.current ? colors.orange : 'rgba(17,17,19,0.35)'} />
      </View>
      <Heading size={wide ? 72 : 52} tabular>
        {pad2(data.streak.current)}
        <Span color={colors.accent}>d</Span>
      </Heading>
      <Text size={13} color="rgba(17,17,19,0.7)">
        {data.streak.activeToday ? 'You learned today — nice.' : data.streak.current ? 'Learn today to keep it alive.' : 'Finish a lesson to start one.'}
      </Text>
    </Card>
  );

  const masteredTile = (
    <Card tone="mint" reveal={3} style={{ width: w(1), ...h(rowH ?? 168), justifyContent: 'space-between', gap: 12 }}>
      <View style={styles.row}>
        <LiveDot />
        <Eyebrow color="rgba(17,17,19,0.6)">Skills mastered</Eyebrow>
      </View>
      <View>
        <Heading size={wide ? 64 : 48} tabular>
          {pad2(data.totalMastered)}
        </Heading>
        <Text size={13} color="rgba(17,17,19,0.6)" style={{ marginTop: 6 }}>
          {data.weekly.mastered} this week · {data.courses.length} course{data.courses.length === 1 ? '' : 's'}
        </Text>
      </View>
    </Card>
  );

  const activityTile = (
    <Card
      reveal={2}
      style={{ width: w(wide ? 1 : 2), ...(wide ? { height: rowH! * 2 + gap } : {}), justifyContent: 'space-between', gap: 20 }}
      decoration={<DotPattern size={16} r={1.5} color="rgba(17,17,19,0.12)" style={{ opacity: 0.5 }} />}
    >
      <Eyebrow>Last 4 weeks</Eyebrow>
      <View style={{ alignItems: wide ? 'center' : 'flex-start' }}>
        <ActivityGrid days={data.activity} cell={wide ? 22 : 30} />
      </View>
      <View style={{ gap: 8 }}>
        {[
          ['Replies to tutor', data.weekly.messages],
          ['Assessments', data.weekly.attempts],
        ].map(([k, v]) => (
          <View key={k as string} style={[styles.row, { justifyContent: 'space-between' }]}>
            <Text size={13} color={colors.muted}>
              {k} · 7d
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
    <Card tone="sky" reveal={4} style={{ width: w(2), gap: 12 }}>
      <Eyebrow color="rgba(17,17,19,0.6)">Your courses</Eyebrow>
      <Heading size={28}>No courses yet</Heading>
      <Text color="rgba(17,17,19,0.7)">Start a course and it will appear here with your progress and next lesson.</Text>
      <PillButton label="Browse courses" icon="arrowUpRight" onPress={() => router.push('/courses')} />
    </Card>
  );

  const openTile = (
    <Card reveal={5} style={{ width: w(2), gap: 14 }}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>Pick up where you left off</Eyebrow>
        <Eyebrow>{pad2(data.openSessions.length)} open</Eyebrow>
      </View>
      {data.openSessions.length === 0 ? (
        <Text color={colors.muted}>No unfinished lessons. Start your next one from a course roadmap.</Text>
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
                {o.courseTitle} · {phaseLabel[o.phase]}
              </Eyebrow>
            </View>
            <PillButton label="Resume" size="sm" variant="secondary" onPress={() => router.push(`/learn/${o.courseSlug}/${o.nodeId}`)} />
          </View>
        ))
      )}
    </Card>
  );

  const attemptsTile = (
    <Card reveal={6} style={{ width: w(wide ? 3 : 2), gap: 14 }} padding={wide ? 28 : 20}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>Recent assessments</Eyebrow>
        <PillButton label="All progress" size="sm" variant="secondary" onPress={() => router.push('/progress')} />
      </View>
      {data.recentAttempts.length === 0 ? (
        <Text color={colors.muted}>No assessments yet. Every lesson ends with a short one — your scores will be listed here.</Text>
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
                {a.courseTitle} · {a.passed ? 'Mastered' : 'Review needed'} · {new Date(a.createdAt).toLocaleDateString()}
              </Eyebrow>
            </View>
          </View>
        ))
      )}
    </Card>
  );

  const exploreTile =
    data.explore.length > 0 ? (
      <CourseCard course={data.explore[0]} reveal={7} style={{ width: w(wide ? 1 : 2) }} />
    ) : (
      <Card tone="peach" reveal={7} onPress={() => router.push('/courses')} accessibilityLabel="Explore courses" style={{ width: w(wide ? 1 : 2), minHeight: 180, justifyContent: 'space-between' }}>
        <View style={[styles.row, { justifyContent: 'space-between' }]}>
          <Eyebrow color="rgba(17,17,19,0.6)">Explore</Eyebrow>
          <ArrowBadge size={34} />
        </View>
        <View>
          <Heading size={28}>More courses soon</Heading>
          <Text size={13} color="rgba(17,17,19,0.7)" style={{ marginTop: 6 }}>
            You&apos;re enrolled in every course available today.
          </Text>
        </View>
      </Card>
    );

  return (
    <Screen onRefresh={refetch} refreshing={isRefetching}>
      {!wide ? <PageHeader label="Home" eyebrow={today()} title={`Hi, ${first}`} aside={<Avatar name={data.user.displayName} size={40} />} /> : null}
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
