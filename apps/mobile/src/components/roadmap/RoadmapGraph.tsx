import { useRouter } from 'expo-router';
import { useMemo, useState } from 'react';
import { Platform, Pressable, StyleSheet, View } from 'react-native';
import Svg, { Path } from 'react-native-svg';

import { PulseDot } from '@/components/ui/Decor';
import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { LevelSummary, RoadmapNode } from '@/lib/api';
import { colors, radius, shadow } from '@/theme/tokens';
import { edgePath, layoutRoadmap, type Box } from './layout';

const unlocked = (n: RoadmapNode) => n.status === 'available' || n.status === 'in_progress' || n.status === 'mastered';

function NodeCard({ node, box, next }: { node: RoadmapNode; box: Box; next: boolean }) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  const s = node.status;
  const bg = s === 'mastered' ? colors.mint : s === 'in_progress' ? colors.lavender : s === 'coming_soon' ? 'rgba(255,255,255,0.55)' : colors.card;
  return (
    <Pressable
      onPress={() => router.push(`/nodes/${node.id}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={`${node.title}, ${node.level}, ${s.replace('_', ' ')}`}
      style={[
        styles.node,
        {
          left: box.x,
          top: box.y,
          width: box.w,
          height: box.h,
          backgroundColor: bg,
          borderColor: s === 'available' || next ? colors.accent : colors.line,
          borderWidth: s === 'available' || next ? 1.5 : 1,
          borderStyle: s === 'coming_soon' ? 'dashed' : 'solid',
          opacity: s === 'locked' ? 0.6 : 1,
          boxShadow: s === 'coming_soon' ? 'none' : hovered ? shadow.cardHover : shadow.card,
          transform: [{ translateY: hovered ? -2 : 0 }],
        },
        Platform.OS === 'web' && ({ transitionProperty: 'box-shadow, transform', transitionDuration: '400ms' } as object),
      ]}
    >
      <View style={styles.nodeTop}>
        <Eyebrow color={s === 'mastered' ? colors.emeraldDeep : s === 'available' || s === 'in_progress' ? colors.accent : colors.muted}>
          {node.level} · {s === 'coming_soon' ? 'Soon' : s === 'in_progress' ? 'In progress' : s}
        </Eyebrow>
        {s === 'mastered' ? (
          <View style={styles.check}>
            <Icon name="check" size={12} color={colors.white} strokeWidth={2.5} />
          </View>
        ) : s === 'locked' ? (
          <Icon name="lock" size={14} color={colors.muted} />
        ) : null}
      </View>
      <Text size={14} weight="medium" tight numberOfLines={2} color={s === 'coming_soon' ? colors.muted : colors.ink}>
        {node.title}
      </Text>
      {next ? (
        <View style={styles.pulse} pointerEvents="none">
          <PulseDot size={10} rings={false} />
        </View>
      ) : null}
    </Pressable>
  );
}

// Shared SVG node-and-edge roadmap (works on Android, iOS and Web).
export function RoadmapGraph({ nodes, levels, width, wide, nextNodeId }: { nodes: RoadmapNode[]; levels: LevelSummary[]; width: number; wide: boolean; nextNodeId: string | null }) {
  const layout = useMemo(() => layoutRoadmap(nodes, levels, width, wide), [nodes, levels, width, wide]);
  const byId = useMemo(() => new Map(nodes.map((n) => [n.id, n])), [nodes]);

  const edges = nodes.flatMap((n) =>
    n.prerequisiteIds.map((p) => {
      const from = byId.get(p)!;
      const active = from.status === 'mastered' && unlocked(n);
      return { key: `${p}->${n.id}`, d: edgePath(layout.boxes[p], layout.boxes[n.id]), active, soon: n.status === 'coming_soon' };
    }),
  );

  return (
    <View style={{ width, height: layout.height }}>
      <Svg width={width} height={layout.height} style={StyleSheet.absoluteFill}>
        {edges
          .sort((a, b) => Number(a.active) - Number(b.active))
          .map((e) => (
            <Path
              key={e.key}
              d={e.d}
              fill="none"
              stroke={e.active ? colors.accent : colors.lineStrong}
              strokeOpacity={e.active ? 0.7 : 1}
              strokeWidth={e.active ? 2 : 1.25}
              strokeDasharray={e.soon ? '4 5' : undefined}
            />
          ))}
      </Svg>
      {layout.headers.map((h) => (
        <View key={h.level} style={[styles.header, { top: h.y }]}>
          <View style={styles.levelPill}>
            <Text size={12} weight="semibold" color={colors.white}>
              {h.level}
            </Text>
          </View>
          <Heading size={wide ? 22 : 18}>{h.title}</Heading>
          <Eyebrow style={{ marginLeft: 'auto' }}>
            {h.mastered}/{h.total} mastered
          </Eyebrow>
        </View>
      ))}
      {nodes.map((n) => (
        <NodeCard key={n.id} node={n} box={layout.boxes[n.id]} next={n.id === nextNodeId} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  node: {
    position: 'absolute',
    borderRadius: radius.xxl + 4,
    padding: 14,
    justifyContent: 'space-between',
  },
  nodeTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 6 },
  check: { width: 18, height: 18, borderRadius: 9, backgroundColor: colors.emerald, alignItems: 'center', justifyContent: 'center' },
  pulse: { position: 'absolute', right: -10, top: -10 },
  header: { position: 'absolute', left: 0, right: 0, height: 40, flexDirection: 'row', alignItems: 'center', gap: 10 },
  levelPill: { backgroundColor: colors.ink, borderRadius: radius.pill, paddingHorizontal: 10, paddingVertical: 4 },
});
