import { Redirect, useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { ComingSoonCard, CourseCard } from '@/components/CourseCard';
import { Headline } from '@/components/Headline';
import { ErrorState, LoadingState, pad2, SectionHeader } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern, GradientRing, LiveDot, MeshBlob, PulseDot } from '@/components/ui/Decor';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Span, Text } from '@/components/ui/Text';
import { useAuth } from '@/lib/auth';
import { useCourses } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';

const steps = [
  { n: '01', title: 'Follow the roadmap', text: 'Every skill sits on a visible path with prerequisites, from first steps to the finish line.' },
  { n: '02', title: 'Learn with an AI tutor', text: 'Short explanations, examples and practice in a chat — with hints when you get stuck.' },
  { n: '03', title: 'Prove it, unlock the next', text: 'A short assessment scores each objective. Pass it and the next skills open up.' },
];

export default function Landing() {
  const router = useRouter();
  const status = useAuth((s) => s.status);
  const courses = useCourses();
  const { wide, gap, contentWidth, cols, width } = useLayout();
  const w = (n: number) => span(contentWidth, gap, cols, n);

  if (status === 'authenticated') return <Redirect href="/dashboard" />;

  const list = courses.data ?? [];
  const lessons = list.reduce((n, c) => n + c.readyNodes, 0);
  const skills = list.reduce((n, c) => n + c.totalNodes, 0);
  const rowH = wide ? 268 : undefined;

  const hero = (
    <Card
      reveal={0}
      padding={wide ? 32 : 24}
      style={{ width: w(2), minHeight: wide ? rowH! * 2 + gap : undefined, justifyContent: 'space-between', gap: 40 }}
      decoration={
        <>
          <MeshBlob color={colors.lavender} size={340} style={{ right: -110, top: -110 }} />
          <MeshBlob color={colors.peach} size={280} style={{ right: 20, bottom: -130 }} delay={2500} />
        </>
      }
    >
      <View style={styles.row}>
        <GradientRing size={48} />
        <View>
          <Text weight="medium" tight>
            FluentPath
          </Text>
          <Eyebrow>Roadmap-first learning</Eyebrow>
        </View>
      </View>
      <View>
        <Headline size={wide ? (width >= 1024 ? 60 : 52) : 42} before="Learn anything with a" gradient="clear roadmap" after=" and an AI tutor." />
        <Text color={colors.muted} size={wide ? 18 : 15} style={{ marginTop: 20, maxWidth: 440 }}>
          Pick a course, follow its skill roadmap, learn each step in an interactive lesson and prove it before moving on. Your progress lives on your own dashboard.
        </Text>
        <View style={[styles.row, { marginTop: 28, flexWrap: 'wrap', gap: 8 }]}>
          <PillButton label="Get started — it's free" icon="arrowUpRight" onPress={() => router.push('/register')} />
          <PillButton label="Browse courses" variant="secondary" onPress={() => router.push('/courses')} />
        </View>
      </View>
    </Card>
  );

  const coursesTile = (
    <Card tone="lavender" reveal={1} style={{ width: w(1), height: rowH, minHeight: wide ? undefined : 170, justifyContent: 'space-between' }}>
      <Eyebrow color="rgba(17,17,19,0.6)">Courses</Eyebrow>
      <Heading size={wide ? 72 : 52} tabular>
        {pad2(list.length)}
        <Span color={colors.accent}>+</Span>
      </Heading>
      <Text size={13} color="rgba(17,17,19,0.7)">
        English first — more roadmaps on the way
      </Text>
    </Card>
  );

  const liveTile = (
    <Card tone="mint" reveal={3} style={{ width: w(1), height: rowH, minHeight: wide ? undefined : 170, justifyContent: 'space-between' }}>
      <View style={styles.row}>
        <LiveDot />
        <Eyebrow color="rgba(17,17,19,0.6)">Ready to learn</Eyebrow>
      </View>
      <View>
        <Heading size={wide ? 44 : 34} tabular>
          {pad2(lessons)} lessons
        </Heading>
        <Text size={13} color="rgba(17,17,19,0.6)" style={{ marginTop: 6 }}>
          across {skills} roadmap skills
        </Text>
      </View>
    </Card>
  );

  const pathTile = (
    <Card
      reveal={2}
      onPress={() => router.push(list[0] ? `/courses/${list[0].slug}` : '/courses')}
      accessibilityLabel="See a roadmap"
      style={{ width: w(wide ? 1 : 2), height: wide ? rowH! * 2 + gap : 260, justifyContent: 'space-between' }}
      decoration={<DotPattern size={16} r={1.5} color="rgba(17,17,19,0.16)" style={{ opacity: 0.55 }} />}
    >
      <Eyebrow>Your path</Eyebrow>
      <View style={{ alignItems: 'center' }}>
        <PulseDot size={wide ? 16 : 12} />
      </View>
      <View>
        <Text weight="medium" size={wide ? 22 : 18} tight>
          See every step before you start
        </Text>
        <Eyebrow style={{ marginTop: 8 }}>Open the English roadmap →</Eyebrow>
      </View>
    </Card>
  );

  return (
    <Screen onRefresh={courses.refetch} refreshing={courses.isRefetching}>
      {wide ? (
        <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
          {hero}
          <View style={{ gap }}>
            {coursesTile}
            {liveTile}
          </View>
          {pathTile}
        </View>
      ) : (
        <View style={{ gap }}>
          {hero}
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {coursesTile}
            {liveTile}
          </View>
          {pathTile}
        </View>
      )}

      <View style={{ marginTop: wide ? 96 : 64 }}>
        <SectionHeader wide={wide} label="Courses" title={<Headline size={wide ? 60 : 36} before="Choose a" gradient="roadmap" after="." />} description="Each course is a roadmap of skills. Browse freely — sign up when you're ready to learn and track progress." />
        {courses.isPending ? (
          <LoadingState label="Loading courses" />
        ) : courses.isError ? (
          <ErrorState title="Can't load courses" message={courses.error.message} onRetry={courses.refetch} />
        ) : (
          <View style={[styles.row, { gap, flexWrap: 'wrap', alignItems: 'stretch' }]}>
            {list.map((c, i) => (
              <CourseCard key={c.slug} course={c} reveal={i} style={{ width: wide ? span(contentWidth, gap, 3, 1) : contentWidth }} />
            ))}
            <ComingSoonCard style={{ width: wide ? span(contentWidth, gap, 3, 1) : contentWidth }} />
          </View>
        )}
      </View>

      <View style={{ marginTop: wide ? 96 : 64, flexDirection: wide ? 'row' : 'column', gap }}>
        {steps.map((s, i) => (
          <Card key={s.n} reveal={i} style={{ flex: wide ? 1 : undefined, gap: 14 }}>
            <Text size={13} color={colors.accent} style={{ fontFamily: 'GeistMono_400Regular' }}>
              {s.n}
            </Text>
            <Heading size={24}>{s.title}</Heading>
            <Text color={colors.muted}>{s.text}</Text>
          </Card>
        ))}
      </View>

      <Card
        tone="ink"
        padding={wide ? 56 : 28}
        spotlight={false}
        style={{ marginTop: wide ? 96 : 64, alignItems: 'center' }}
        decoration={
          <>
            <MeshBlob color="rgba(91,76,245,0.6)" size={420} style={{ left: -80, top: -130 }} />
            <MeshBlob color="rgba(192,76,245,0.4)" size={440} style={{ right: -40, bottom: -180 }} delay={2000} />
            <MeshBlob color="rgba(245,132,76,0.3)" size={280} style={{ left: '35%', bottom: -90 }} delay={4000} />
          </>
        }
      >
        <View style={styles.inkPill}>
          <View style={{ width: 6, height: 6, borderRadius: 3, backgroundColor: colors.white }} />
          <Eyebrow color="rgba(255,255,255,0.7)">Free while in beta</Eyebrow>
        </View>
        <Heading size={wide ? 64 : 40} color={colors.white} style={{ textAlign: 'center', marginTop: 24, maxWidth: 720 }}>
          Start your first lesson today.
        </Heading>
        <Text color="rgba(255,255,255,0.7)" size={wide ? 18 : 15} style={{ textAlign: 'center', marginTop: 16, maxWidth: 440 }}>
          Create an account and FluentPath keeps your roadmap, lessons and scores on your dashboard.
        </Text>
        <View style={[styles.row, { marginTop: 32, flexWrap: 'wrap', justifyContent: 'center', gap: 10 }]}>
          <PillButton label="Create free account" icon="arrowUpRight" variant="accent" onPress={() => router.push('/register')} />
          <PillButton label="Log in" variant="secondary" onPress={() => router.push('/login')} />
        </View>
      </Card>
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  inkPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.15)',
    backgroundColor: 'rgba(255,255,255,0.05)',
    borderRadius: radius.pill,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
});
