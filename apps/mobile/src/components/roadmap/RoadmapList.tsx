import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { ProgressBar, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { LevelSummary, RoadmapNode } from '@/lib/api';
import { useT } from '@/i18n';
import { colors, radius } from '@/theme/tokens';

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
      accessibilityLabel={`${node.title}, ${t(`status.${node.status}`)}`}
      style={[styles.row, hovered && { backgroundColor: colors.canvas }]}
    >
      <View style={{ flex: 1, gap: 4 }}>
        <Text weight="medium" tight color={node.status === 'coming_soon' || node.status === 'locked' ? colors.muted : colors.ink}>
          {node.title}
        </Text>
        <Text size={13} color={colors.muted} numberOfLines={1}>
          {node.summary}
        </Text>
      </View>
      <StatusChip status={node.status} />
      <Icon name="chevronRight" size={16} color={colors.muted} />
    </Pressable>
  );
}

// Accessible list alternative to the graph, grouped by level.
export function RoadmapList({ course, nodes, levels }: { course: string; nodes: RoadmapNode[]; levels: LevelSummary[] }) {
  return (
    <View style={{ gap: 12 }}>
      {levels.map((l, i) => (
        <Card key={l.level} reveal={i} padding={0} spotlight={false}>
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
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
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
