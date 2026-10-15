import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { GoalBadge, ProgressBar, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { Course, LevelSummary, RoadmapNode } from '@/lib/api';
import { useT } from '@/i18n';
import { colors, radius } from '@/theme/tokens';
import type { Branch } from './layout';

function Row({ course, node }: { course: string; node: RoadmapNode }) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  const t = useT();
  return (
    <Pressable
      onPress={() => router.push(`/courses/${course}/${node.id}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={`${node.title}, ${t(`status.${node.status}`)}${node.recommended ? `, ${t('onboarding.recommended')}` : ''}`}
      style={[styles.row, hovered && { backgroundColor: colors.canvas }]}
    >
      <View style={{ flex: 1, gap: 4 }}>
        <Text weight="medium" tight color={node.status === 'coming_soon' || node.status === 'locked' ? colors.muted : colors.ink}>
          {node.title}
        </Text>
        <Text size={13} color={colors.muted} numberOfLines={1}>
          {node.summary}
        </Text>
        {node.recommended && node.status !== 'mastered' ? <GoalBadge /> : null}
      </View>
      <StatusChip status={node.status} />
      <Icon name="chevronRight" size={16} color={colors.muted} />
    </Pressable>
  );
}

function TrackRow({ track }: { track: Course }) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={() => router.push(`/courses/${track.slug}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      style={[styles.row, hovered && { backgroundColor: colors.canvas }]}
    >
      <View style={{ flex: 1, gap: 4 }}>
        <Text weight="medium" tight>
          {track.title}
        </Text>
        <Text size={13} color={colors.muted} numberOfLines={1}>
          {track.tagline}
        </Text>
      </View>
      {track.status === 'coming_soon' ? <StatusChip status="coming_soon" /> : null}
      <Icon name="arrowUpRight" size={16} color={colors.accent} />
    </Pressable>
  );
}

// The side tracks that branch off after a level, between it and the next.
function BranchCard({ branch, next }: { branch: Branch; next?: string }) {
  const t = useT();
  return (
    <Card padding={0} spotlight={false} style={styles.branch}>
      <View style={styles.branchHead}>
        <Icon name="route" size={18} color={colors.accent} />
        <View style={{ flex: 1, gap: 2 }}>
          <Eyebrow color={colors.accent}>{t('course.branchAfter', { level: branch.level })}</Eyebrow>
          <Text weight="medium" size={17} tight>
            {t('course.branchTitle')}
          </Text>
          {next ? (
            <Text size={13} color={colors.muted}>
              {t('course.branchBack', { level: next })}
            </Text>
          ) : null}
        </View>
      </View>
      {branch.tracks.map((tr) => (
        <TrackRow key={tr.slug} track={tr} />
      ))}
    </Card>
  );
}

// Accessible list alternative to the graph, grouped by level.
export function RoadmapList({ course, nodes, levels, branches = [] }: { course: string; nodes: RoadmapNode[]; levels: LevelSummary[]; branches?: Branch[] }) {
  return (
    <View style={{ gap: 12 }}>
      {levels.map((l, i) => (
        <View key={l.level} style={{ gap: 12 }}>
          <Card reveal={i} padding={0} spotlight={false}>
            <View style={styles.head}>
              <View style={styles.levelPill}>
                <Text size={12} weight="semibold" color={colors.white}>
                  {l.level}
                </Text>
              </View>
              <View style={{ flex: 1 }}>
                <Heading size={22}>{l.title}</Heading>
                <Text size={13} color={colors.muted} style={{ marginTop: 4 }}>
                  {l.description}
                </Text>
              </View>
              <View style={{ width: 96, gap: 6 }}>
                <Eyebrow style={{ textAlign: 'right' }}>
                  {l.mastered}/{l.total}
                </Eyebrow>
                <ProgressBar value={l.total ? (l.mastered / l.total) * 100 : 0} color={colors.emerald} height={4} />
              </View>
            </View>
            {nodes
              .filter((n) => n.level === l.level)
              .map((n) => (
                <Row key={n.id} course={course} node={n} />
              ))}
          </Card>
          {branches
            .filter((b) => b.level === l.level)
            .map((b) => (
              <BranchCard key={b.anchor} branch={b} next={levels[i + 1]?.level} />
            ))}
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  branch: { marginLeft: 24, borderColor: colors.accent, borderStyle: 'dashed' },
  branchHead: { flexDirection: 'row', alignItems: 'flex-start', gap: 12, padding: 20 },
  head: { flexDirection: 'row', alignItems: 'flex-start', gap: 14, padding: 20 },
  levelPill: { backgroundColor: colors.ink, borderRadius: radius.pill, paddingHorizontal: 10, paddingVertical: 4, marginTop: 2 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderTopWidth: 1,
    borderTopColor: colors.line,
  },
});
