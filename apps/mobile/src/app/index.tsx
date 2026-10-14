import { useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { Headline } from '@/components/Headline';
import { ErrorState, LoadingState, pad2, ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern, GradientRing, LiveDot, MeshBlob, PulseDot } from '@/components/ui/Decor';
import { ArrowBadge, PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Span, Text } from '@/components/ui/Text';
import { useProgress, useRoadmap } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';

export default function Dashboard() {
  const router = useRouter();
  const roadmap = useRoadmap();
  const progress = useProgress();
  const { wide, gap, contentWidth, cols, width } = useLayout();
  const w = (n: number) => span(contentWidth, gap, cols, n);

  if (roadmap.isPending) {
    return (
      <Screen>
        <LoadingState label="Loading your path" />
      </Screen>
    );
  }
  if (roadmap.isError) {
    return (
      <Screen>
        <ErrorState title="Can't load your roadmap" message={roadmap.error.message} onRetry={() => roadmap.refetch()} />
      </Screen>
    );
  }

  const { learner, summary, nodes, nextNodeId } = roadmap.data;
  const next = nodes.find((n) => n.id === nextNodeId) ?? null;
  const pct = summary.targetNodes ? Math.round((summary.targetMastered / summary.targetNodes) * 100) : 0;
  const levelInfo = summary.levels.find((l) => l.level === learner.currentLevel);
  const targetInfo = summary.levels.find((l) => l.level === learner.targetLevel);
  const rowH = wide ? 268 : 176;
  // Fixed heights keep the wide bento aligned; phones grow with content.
  const h = (v: number | undefined) => (wide ? { height: v } : { minHeight: v });
  const heroH = wide ? rowH * 2 + gap : undefined;
  const goNext = () => next && router.push(`/learn/${next.id}`);

  const hero = (
    <Card
      reveal={0}
      style={[{ width: w(2), minHeight: heroH, justifyContent: 'space-between', gap: 40 }]}
      padding={wide ? 32 : 24}
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
          <Eyebrow>
            English · {learner.currentLevel} → {learner.targetLevel}
          </Eyebrow>
        </View>
      </View>
      <View>
        <Eyebrow style={{ marginBottom: 14 }}>Hi {learner.displayName}, your path to {learner.targetLevel}</Eyebrow>
        <Headline size={wide ? (width >= 1024 ? 60 : 52) : 42} before="A clear path to" gradient="fluent English," after=" one skill at a time." />
        <Text color={colors.muted} size={wide ? 18 : 15} style={{ marginTop: 20, maxWidth: 440 }}>
          Follow a roadmap from A0 foundations to B2 professional fluency. Learn each skill with an AI tutor, prove it in a short assessment, and unlock
          what&apos;s next.
        </Text>
        <View style={[styles.row, { marginTop: 28, flexWrap: 'wrap', gap: 8 }]}>
          {next ? <PillButton label={next.status === 'in_progress' ? 'Continue lesson' : 'Start next lesson'} icon="arrowUpRight" onPress={goNext} /> : null}
          <PillButton label="View roadmap" variant="secondary" onPress={() => router.push('/roadmap')} />
        </View>
      </View>
    </Card>
  );

  const progressTile = (
    <Card tone="lavender" reveal={1} style={{ width: w(1), ...h(rowH), justifyContent: 'space-between', gap: 16 }}>
      <Eyebrow color="rgba(17,17,19,0.6)">Progress to {learner.targetLevel}</Eyebrow>
      <Heading size={wide ? 72 : 52} tabular>
        {pct}
        <Span color={colors.accent}>%</Span>
      </Heading>
      <Text size={13} color="rgba(17,17,19,0.7)">
        {summary.targetMastered} of {summary.targetNodes} skills mastered on the way to {learner.targetLevel}
      </Text>
    </Card>
  );

  const levelTile = (
    <Card tone="mint" reveal={3} style={{ width: w(1), ...h(rowH), justifyContent: 'space-between', gap: 16 }}>
      <View style={styles.row}>
        <LiveDot />
        <Eyebrow color="rgba(17,17,19,0.6)">Current level</Eyebrow>
      </View>
      <View>
        <Heading size={wide ? 30 : 24}>
          {learner.currentLevel} {levelInfo?.title}
        </Heading>
        <Text size={13} color="rgba(17,17,19,0.6)" style={{ marginTop: 6 }}>
          Goal: {learner.targetLevel} · {targetInfo?.title}
        </Text>
      </View>
    </Card>
  );

  const nextTile = (
    <Card
      reveal={2}
      onPress={next ? goNext : undefined}
      accessibilityLabel={next ? `Next up: ${next.title}` : undefined}
      style={{ width: w(wide ? 1 : 2), ...h(wide ? heroH : 300), justifyContent: 'space-between', gap: 16 }}
      decoration={<DotPattern size={16} r={1.5} color="rgba(17,17,19,0.16)" style={{ opacity: 0.55 }} />}
    >
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>Next up</Eyebrow>
        {next ? <ArrowBadge /> : null}
      </View>
      <View style={{ alignItems: 'center' }}>
        <PulseDot size={wide ? 16 : 12} />
      </View>
      {next ? (
        <View>
          <Text weight="medium" size={wide ? 22 : 18} tight>
            {next.title}
          </Text>
          <Text size={13} color={colors.muted} style={{ marginTop: 4 }} numberOfLines={2}>
            {next.summary}
          </Text>
          <Eyebrow style={{ marginTop: 10 }}>
            {next.level} · {next.objectives.length} objectives
          </Eyebrow>
        </View>
      ) : (
        <View>
          <Text weight="medium" size={20} tight>
            All ready lessons mastered
          </Text>
          <Eyebrow style={{ marginTop: 8 }}>New lessons are coming soon</Eyebrow>
        </View>
      )}
    </Card>
  );

  const levelsTile = (
    <Card reveal={4} style={{ width: w(2), minHeight: wide ? 200 : undefined, gap: 16 }} onPress={() => router.push('/roadmap')} accessibilityLabel="Open roadmap">
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>Roadmap levels</Eyebrow>
        <Eyebrow>{summary.totalNodes} skills</Eyebrow>
      </View>
      <View style={{ gap: 10 }}>
        {summary.levels.map((l) => {
          const isCurrent = l.level === learner.currentLevel;
          const isTarget = l.level === learner.targetLevel;
          return (
            <View key={l.level} style={[styles.row, { gap: 12 }]}>
              <View style={[styles.levelBadge, isCurrent && { backgroundColor: colors.ink, borderColor: colors.ink }]}>
                <Text size={12} weight="medium" color={isCurrent ? colors.white : colors.ink}>
                  {l.level}
                </Text>
              </View>
              <View style={{ flex: 1, gap: 6 }}>
                <View style={[styles.row, { justifyContent: 'space-between' }]}>
                  <Text size={13} weight={isCurrent ? 'medium' : 'regular'}>
                    {l.title}
                    {isTarget ? <Span color={colors.accent}>{'  · goal'}</Span> : null}
                  </Text>
                  <Text size={12} color={colors.muted} tabular>
                    {l.mastered}/{l.total}
                  </Text>
                </View>
                <ProgressBar value={l.total ? (l.mastered / l.total) * 100 : 0} color={isCurrent ? colors.accent : colors.emerald} height={4} />
              </View>
            </View>
          );
        })}
      </View>
    </Card>
  );

  const weekly = progress.data?.weekly;
  const weekTile = (
    <Card reveal={5} style={{ width: w(1), minHeight: wide ? 200 : 180, justifyContent: 'space-between' }}>
      <Eyebrow>This week</Eyebrow>
      <View style={{ gap: 10 }}>
        {[
          ['Replies to tutor', weekly?.messages],
          ['Assessments', weekly?.attempts],
          ['Skills mastered', weekly?.mastered],
        ].map(([label, value]) => (
          <View key={label as string} style={[styles.row, { justifyContent: 'space-between' }]}>
            <Text size={13} color={colors.muted}>
              {label}
            </Text>
            <Text size={18} weight="semibold" tabular tight>
              {value ?? '–'}
            </Text>
          </View>
        ))}
      </View>
    </Card>
  );

  const roadmapTile = (
    <Card tone="peach" reveal={6} onPress={() => router.push('/roadmap')} accessibilityLabel="Open roadmap" style={{ width: w(1), minHeight: wide ? 200 : 180, justifyContent: 'space-between' }}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow color="rgba(17,17,19,0.6)">Roadmap</Eyebrow>
        <ArrowBadge />
      </View>
      <View>
        <Heading size={wide ? 64 : 48} tabular>
          {pad2(summary.readyNodes)}
        </Heading>
        <Text size={13} color="rgba(17,17,19,0.7)" style={{ marginTop: 4 }}>
          lessons ready · {summary.totalNodes - summary.readyNodes} coming soon
        </Text>
      </View>
    </Card>
  );

  const attempts = progress.data?.recentAttempts ?? [];
  const recentTile = (
    <Card reveal={7} style={{ width: w(wide ? 3 : 2), gap: 16 }} padding={wide ? 32 : 24}>
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>Recent assessments</Eyebrow>
        <PillButton label="All progress" size="sm" variant="secondary" onPress={() => router.push('/progress')} />
      </View>
      {attempts.length === 0 ? (
        <Text color={colors.muted}>No assessments yet. Finish your first lesson and your scores will show up here.</Text>
      ) : (
        attempts.slice(0, 4).map((a) => (
          <View key={a.id} style={[styles.row, styles.attemptRow]}>
            <View style={[styles.scorePill, { backgroundColor: a.passed ? colors.mint : colors.peach }]}>
              <Text size={14} weight="semibold" tabular>
                {a.score}
              </Text>
            </View>
            <View style={{ flex: 1 }}>
              <Text weight="medium" tight>
                {a.title}
              </Text>
              <Eyebrow>{a.passed ? 'Mastered' : 'Review needed'} · {new Date(a.createdAt).toLocaleDateString()}</Eyebrow>
            </View>
          </View>
        ))
      )}
    </Card>
  );

  const masteredTile = (
    <Card tone="sky" reveal={8} style={{ width: w(wide ? 1 : 2), minHeight: 160, justifyContent: 'space-between' }}>
      <Eyebrow color="rgba(17,17,19,0.6)">Skills mastered</Eyebrow>
      <View>
        <Heading size={wide ? 64 : 48} tabular>
          {pad2(summary.masteredNodes)}
        </Heading>
        <Eyebrow color="rgba(17,17,19,0.6)" style={{ marginTop: 6 }}>
          of {summary.totalNodes} on the full A0–C2 path
        </Eyebrow>
      </View>
    </Card>
  );

  return (
    <Screen onRefresh={() => Promise.all([roadmap.refetch(), progress.refetch()])} refreshing={roadmap.isRefetching}>
      {wide ? (
        <View style={{ gap }}>
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {hero}
            <View style={{ gap }}>
              {progressTile}
              {levelTile}
            </View>
            {nextTile}
          </View>
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {levelsTile}
            {weekTile}
            {roadmapTile}
          </View>
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {recentTile}
            {masteredTile}
          </View>
        </View>
      ) : (
        <View style={{ gap }}>
          {hero}
          <View style={[styles.row, { gap }]}>
            {progressTile}
            {levelTile}
          </View>
          {nextTile}
          {levelsTile}
          <View style={[styles.row, { gap, alignItems: 'stretch' }]}>
            {weekTile}
            {roadmapTile}
          </View>
          {recentTile}
          {masteredTile}
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  levelBadge: {
    width: 34,
    height: 28,
    borderRadius: radius.pill,
    borderWidth: 1,
    borderColor: colors.line,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.card,
  },
  attemptRow: { paddingTop: 12, borderTopWidth: 1, borderTopColor: colors.line },
  scorePill: { width: 44, height: 44, borderRadius: radius.pill, alignItems: 'center', justifyContent: 'center' },
});
