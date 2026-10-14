import { useLocalSearchParams, useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { Chip, DotList, ErrorState, LoadingState, pad2, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { GradientRing, MeshBlob } from '@/components/ui/Decor';
import { GradientText } from '@/components/ui/GradientText';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useAuth } from '@/lib/auth';
import { useNode } from '@/lib/queries';
import { colors, fonts } from '@/theme/tokens';

export default function NodeScreen() {
  const { course, node: nodeId } = useLocalSearchParams<{ course: string; node: string }>();
  const router = useRouter();
  const authed = useAuth((s) => s.status === 'authenticated');
  const { data: node, isPending, isError, error, refetch } = useNode(course, nodeId);
  const { wide, gap, contentWidth } = useLayout();

  if (isPending) {
    return (
      <Screen>
        <LoadingState label="Loading lesson" />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState title={error.message.includes('No roadmap node') ? 'Lesson not found' : "Can't load this lesson"} message={error.message} onRetry={refetch} />
      </Screen>
    );
  }

  const label = node.status === 'mastered' ? 'Practise again' : node.openSessionId || node.status === 'in_progress' ? 'Continue lesson' : 'Start lesson';
  const mainW = wide ? span(contentWidth, gap, 3, 2) : contentWidth;
  const sideW = wide ? span(contentWidth, gap, 3, 1) : contentWidth;

  const main = (
    <View style={{ width: mainW, gap }}>
      <Card reveal={0} padding={wide ? 32 : 24} style={{ gap: 28 }} decoration={<MeshBlob color={colors.lavender} size={300} style={{ right: -120, top: -140 }} />}>
        <View style={styles.row}>
          <Crumb onPress={() => router.push(`/courses/${node.course.slug}`)} label={node.course.title} />
          <Eyebrow>/ {node.level}</Eyebrow>
        </View>
        <View style={{ gap: 14 }}>
          <StatusChip status={node.status} />
          <Heading size={wide ? 52 : 38}>{node.title}</Heading>
          <Text color={colors.muted} size={wide ? 18 : 15} style={{ maxWidth: 520 }}>
            {node.summary}
          </Text>
        </View>
        <View style={{ gap: 10 }}>
          {node.canStart && !authed ? (
            <>
              <PillButton
                label="Log in to start"
                icon="arrowUpRight"
                onPress={() => router.push({ pathname: '/login', params: { next: `/courses/${node.course.slug}/${node.id}` } })}
              />
              <Text size={13} color={colors.muted}>
                Lessons and progress need a free account.
              </Text>
            </>
          ) : node.canStart ? (
            <PillButton label={label} icon="arrowUpRight" onPress={() => router.push(`/learn/${node.course.slug}/${node.id}`)} />
          ) : (
            <>
              <PillButton label={node.status === 'coming_soon' ? 'Coming soon' : 'Locked'} icon={node.status === 'locked' ? 'lock' : undefined} disabled />
              <Text size={13} color={colors.muted}>
                {node.blockedReason}
              </Text>
            </>
          )}
        </View>
      </Card>

      <Card reveal={1} padding={wide ? 32 : 24} style={{ gap: 20 }}>
        <Eyebrow>What you&apos;ll be able to do</Eyebrow>
        {node.objectives.map((o, i) => (
          <View key={o.id} style={[styles.objective, i > 0 && styles.divider]}>
            <Text size={13} color={colors.accent} style={{ fontFamily: fonts.mono, width: 28, marginTop: 2 }}>
              {pad2(i + 1)}
            </Text>
            <Text size={wide ? 18 : 16} weight="medium" tight style={{ flex: 1 }}>
              {o.text}
            </Text>
          </View>
        ))}
      </Card>

      <View style={{ flexDirection: wide ? 'row' : 'column', gap }}>
        {[
          { title: 'Prerequisites', items: node.prerequisites, empty: 'None — this is a starting point.' },
          { title: 'Unlocks', items: node.successors, empty: 'This is the end of this branch for now.' },
        ].map((g, i) => (
          <Card key={g.title} reveal={2 + i} style={{ flex: wide ? 1 : undefined, gap: 14 }}>
            <Eyebrow>{g.title}</Eyebrow>
            {g.items.length === 0 ? (
              <Text size={14} color={colors.muted}>
                {g.empty}
              </Text>
            ) : (
              <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
                {g.items.map((n) => (
                  <Chip key={n.id} label={`${n.level} · ${n.title}`} icon={n.status === 'mastered' ? 'check' : n.status === 'locked' ? 'lock' : undefined} onPress={() => router.push(`/courses/${node.course.slug}/${n.id}`)} />
                ))}
              </View>
            )}
          </Card>
        ))}
      </View>
    </View>
  );

  const side = (
    <View style={{ width: sideW, gap }}>
      <Card
        tone="ink"
        reveal={wide ? 1 : 4}
        padding={wide ? 32 : 24}
        style={{ gap: 28 }}
        decoration={<MeshBlob color="rgba(91,76,245,0.5)" size={240} style={{ right: -70, top: -70 }} />}
      >
        <Eyebrow color="rgba(255,255,255,0.6)">Mastery</Eyebrow>
        <View>
          <View style={{ flexDirection: 'row', alignItems: 'flex-end' }}>
            <Heading size={72} color={colors.white} tabular>
              {node.masteryThreshold}
            </Heading>
            <GradientText animated={false} style={{ fontFamily: fonts.semibold, fontSize: 32, lineHeight: 40, letterSpacing: -1.4 }}>
              /100
            </GradientText>
          </View>
          <Text color="rgba(255,255,255,0.7)" style={{ marginTop: 8 }}>
            Score needed in the assessment to master this skill and unlock what comes next.
          </Text>
        </View>
        <View style={{ borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.1)' }}>
          {[
            ['Objectives', pad2(node.objectives.length)],
            ['Key ideas', pad2(node.keyPoints.length)],
            ['Your best score', node.masteryScore != null ? String(node.masteryScore) : '—'],
          ].map(([k, v]) => (
            <View key={k} style={styles.inkRow}>
              <Text size={14} color={colors.white}>
                {k}
              </Text>
              <Text size={12} color="rgba(255,255,255,0.5)" style={{ fontFamily: fonts.mono }}>
                {v}
              </Text>
            </View>
          ))}
        </View>
      </Card>
      {node.keyPoints.length > 0 ? (
        <Card tone="lavender" reveal={wide ? 2 : 5} style={{ gap: 16 }}>
          <View style={[styles.row, { justifyContent: 'space-between' }]}>
            <Eyebrow color="rgba(17,17,19,0.6)">Key ideas</Eyebrow>
            <GradientRing size={32} />
          </View>
          <DotList items={node.keyPoints} />
        </Card>
      ) : null}
    </View>
  );

  return (
    <Screen>
      <View style={{ flexDirection: wide ? 'row' : 'column', gap, alignItems: 'flex-start' }}>
        {main}
        {side}
      </View>
    </Screen>
  );
}

function Crumb({ onPress, label }: { onPress: () => void; label: string }) {
  return (
    <Text size={12} color={colors.muted} onPress={onPress} accessibilityRole="link" style={{ fontFamily: fonts.mono, textTransform: 'uppercase', letterSpacing: 0.7, textDecorationLine: 'underline' }}>
      {label}
    </Text>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  objective: { flexDirection: 'row', gap: 12, paddingTop: 2 },
  divider: { borderTopWidth: 1, borderTopColor: colors.line, paddingTop: 18 },
  inkRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'baseline', paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.1)' },
});
