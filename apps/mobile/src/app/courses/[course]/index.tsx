import { useLocalSearchParams, useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { RoadmapGraph } from '@/components/roadmap/RoadmapGraph';
import { RoadmapList } from '@/components/roadmap/RoadmapList';
import { ErrorState, LoadingState, ProgressBar, SegmentedPill, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern, GradientRing, MeshBlob } from '@/components/ui/Decor';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Span, Text } from '@/components/ui/Text';
import { useEnroll, useRoadmap } from '@/lib/queries';
import { useUI } from '@/lib/store';
import { colors } from '@/theme/tokens';

export default function CourseScreen() {
  const { course: slug } = useLocalSearchParams<{ course: string }>();
  const router = useRouter();
  const { data, isPending, isError, error, refetch, isRefetching } = useRoadmap(slug);
  const enroll = useEnroll();
  const { wide, gap, contentWidth } = useLayout();
  const view = useUI((s) => s.roadmapView);
  const setView = useUI((s) => s.setRoadmapView);
  const pad = wide ? 32 : 16;

  if (isPending) {
    return (
      <Screen>
        <LoadingState label="Loading course" />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState title={error.message.includes('No course') ? 'Course not found' : "Can't load this course"} message={error.message} onRetry={refetch} />
      </Screen>
    );
  }

  const { course, summary, enrollment } = data;
  const next = data.nodes.find((n) => n.id === data.nextNodeId);
  const here = `/courses/${course.slug}`;

  const primary = !data.authenticated ? (
    <>
      <PillButton label="Sign up to start" icon="arrowUpRight" onPress={() => router.push({ pathname: '/register', params: { next: here } })} />
      <PillButton label="Log in" variant="secondary" onPress={() => router.push({ pathname: '/login', params: { next: here } })} />
    </>
  ) : !data.enrolled ? (
    <PillButton label="Start this course" icon="arrowUpRight" loading={enroll.isPending} onPress={() => enroll.mutate(course.slug)} />
  ) : next ? (
    <PillButton label={next.status === 'in_progress' ? `Continue: ${next.title}` : `Next: ${next.title}`} icon="arrowUpRight" onPress={() => router.push(`/learn/${course.slug}/${next.id}`)} />
  ) : (
    <Text color={colors.muted}>Every ready lesson is mastered — new lessons are coming soon.</Text>
  );

  const hero = (
    <Card
      reveal={0}
      padding={wide ? 32 : 24}
      style={{ width: wide ? span(contentWidth, gap, 3, 2) : contentWidth, gap: 24 }}
      decoration={<MeshBlob color={colors.lavender} size={320} style={{ right: -120, top: -140 }} />}
    >
      <View style={styles.row}>
        <GradientRing size={44} />
        <View>
          <Eyebrow>
            {course.category} · {course.levels[0]}–{course.levels[course.levels.length - 1]}
          </Eyebrow>
          <Text size={13} color={colors.muted}>
            {course.readyNodes} lessons ready · {course.totalNodes} skills
          </Text>
        </View>
      </View>
      <View style={{ gap: 10 }}>
        <Heading size={wide ? 60 : 42}>{course.title}</Heading>
        <Text size={wide ? 18 : 15} color={colors.muted} style={{ maxWidth: 560 }}>
          {course.description}
        </Text>
      </View>
      <View style={[styles.row, { flexWrap: 'wrap', gap: 8 }]}>{primary}</View>
      {enroll.isError ? (
        <Text size={13} color={colors.danger}>
          {enroll.error.message}
        </Text>
      ) : null}
    </Card>
  );

  const progressTile = (
    <Card tone={data.enrolled ? 'mint' : 'sky'} reveal={1} style={{ width: wide ? span(contentWidth, gap, 3, 1) : contentWidth, gap: 16, justifyContent: 'space-between' }}>
      <Eyebrow color="rgba(17,17,19,0.6)">{data.enrolled ? 'Your progress' : 'The goal'}</Eyebrow>
      {data.enrolled ? (
        <>
          <Heading size={wide ? 72 : 56} tabular>
            {summary.percent}
            <Span color={colors.accent}>%</Span>
          </Heading>
          <View style={{ gap: 8 }}>
            <ProgressBar value={summary.percent} color={colors.emerald} track="rgba(17,17,19,0.08)" />
            <Text size={13} color="rgba(17,17,19,0.7)">
              Level {enrollment.currentLevel} · {summary.targetMastered} of {summary.targetNodes} skills to {enrollment.targetLevel}
            </Text>
          </View>
        </>
      ) : (
        <>
          <Heading size={wide ? 72 : 56}>{course.targetLevel}</Heading>
          <Text size={13} color="rgba(17,17,19,0.7)">
            {course.tagline}. Levels beyond the goal stay available as optional tracks.
          </Text>
        </>
      )}
    </Card>
  );

  return (
    <Screen onRefresh={refetch} refreshing={isRefetching}>
      <View style={{ flexDirection: wide ? 'row' : 'column', gap, alignItems: 'stretch' }}>
        {hero}
        {progressTile}
      </View>

      <View style={[styles.row, { justifyContent: 'space-between', flexWrap: 'wrap', marginTop: wide ? 56 : 36, marginBottom: 16, gap: 12 }]}>
        <View>
          <Eyebrow>Roadmap</Eyebrow>
          <Heading size={wide ? 36 : 28}>Every skill, in order</Heading>
        </View>
        <SegmentedPill
          value={view}
          onChange={setView}
          options={[
            { value: 'graph', label: 'Graph', icon: 'graph' },
            { value: 'list', label: 'List', icon: 'list' },
          ]}
        />
      </View>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginBottom: 16 }} accessibilityLabel="Legend">
        {(['mastered', 'in_progress', 'available', 'locked', 'coming_soon'] as const).map((s) => (
          <StatusChip key={s} status={s} />
        ))}
      </View>
      {view === 'graph' ? (
        <Card padding={pad} spotlight={false} decoration={<DotPattern size={16} r={1.2} color="rgba(17,17,19,0.10)" />}>
          <RoadmapGraph course={course.slug} nodes={data.nodes} levels={summary.levels} width={contentWidth - pad * 2 - 2} wide={wide} nextNodeId={data.nextNodeId} />
        </Card>
      ) : (
        <RoadmapList course={course.slug} nodes={data.nodes} levels={summary.levels} />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
});
