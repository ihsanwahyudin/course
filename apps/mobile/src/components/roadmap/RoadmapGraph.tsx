import { useRouter } from 'expo-router';
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Modal, Platform, Pressable, ScrollView, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Svg, { Path } from 'react-native-svg';

import { GoalBadge } from '@/components/ui/Bits';
import { PulseDot } from '@/components/ui/Decor';
import { Icon } from '@/components/ui/Icon';
import { IconButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { Course, LevelSummary, RoadmapNode } from '@/lib/api';
import { useT } from '@/i18n';
import { colors, radius, shadow } from '@/theme/tokens';
import { edgePath, layoutRoadmap, type Box, type Branch, type BranchLayout, type GraphLayout } from './layout';

const unlocked = (n: RoadmapNode) => n.status === 'available' || n.status === 'in_progress' || n.status === 'mastered';

function NodeCard({ course, node, box, next, onNavigate }: { course: string; node: RoadmapNode; box: Box; next: boolean; onNavigate?: () => void }) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  const t = useT();
  const s = node.status;
  const bg = s === 'mastered' ? colors.mint : s === 'in_progress' ? colors.lavender : s === 'coming_soon' ? 'rgba(255,255,255,0.55)' : colors.card;
  return (
    <Pressable
      onPress={() => {
        onNavigate?.();
        router.push(`/courses/${course}/${node.id}`);
      }}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={`${node.title}, ${node.level}, ${t(`status.${s}`)}${node.recommended ? `, ${t('onboarding.recommended')}` : ''}`}
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
          {node.level} · {s === 'coming_soon' ? t('status.soon') : t(`status.${s}`)}
        </Eyebrow>
        <View style={styles.icons}>
          {node.recommended && s !== 'mastered' ? <GoalBadge compact /> : null}
          {s === 'mastered' ? (
            <View style={styles.check}>
              <Icon name="check" size={12} color={colors.white} strokeWidth={2.5} />
            </View>
          ) : s === 'locked' ? (
            <Icon name="lock" size={14} color={colors.muted} />
          ) : null}
        </View>
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

function TrackCard({ track, box, level, onNavigate }: { track: Course; box: Box; level: string; onNavigate?: () => void }) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  const t = useT();
  const soon = track.status === 'coming_soon';
  return (
    <Pressable
      onPress={() => {
        onNavigate?.();
        router.push(`/courses/${track.slug}`);
      }}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      accessibilityLabel={`${track.title}, ${t('course.branchAfter', { level })}${soon ? `, ${t('common.comingSoon')}` : ''}`}
      style={[
        styles.node,
        styles.track,
        {
          left: box.x,
          top: box.y,
          width: box.w,
          height: box.h,
          borderStyle: soon ? 'dashed' : 'solid',
          boxShadow: hovered ? shadow.cardHover : 'none',
          transform: [{ translateY: hovered ? -2 : 0 }],
        },
        Platform.OS === 'web' && ({ transitionProperty: 'box-shadow, transform', transitionDuration: '400ms' } as object),
      ]}
    >
      <View style={styles.nodeTop}>
        <Eyebrow color={colors.accent}>
          {track.category}
          {soon ? ` · ${t('status.soon')}` : ''}
        </Eyebrow>
        <Icon name="arrowUpRight" size={14} color={colors.accent} />
      </View>
      <Text size={15} weight="medium" tight numberOfLines={1}>
        {track.title}
      </Text>
    </Pressable>
  );
}

// Head of a branch: where it leaves the main road, which carries on below.
function BranchHead({ lane, nextLevel }: { lane: BranchLayout; nextLevel?: string }) {
  const t = useT();
  return (
    <View pointerEvents="none" style={[styles.branchHead, { left: lane.head.x, top: lane.head.y, width: lane.head.w, height: lane.head.h }]}>
      <Eyebrow>{t('course.branchAfter', { level: lane.level })}</Eyebrow>
      <Text size={17} weight="medium" tight numberOfLines={1}>
        {t('course.branchTitle')}
      </Text>
      {nextLevel ? (
        <Text size={12} color={colors.muted} numberOfLines={1}>
          {t('course.branchBack', { level: nextLevel })}
        </Text>
      ) : null}
    </View>
  );
}

type Canvas = { course: string; nodes: RoadmapNode[]; levels: LevelSummary[]; layout: GraphLayout; width: number; nextNodeId: string | null; onNavigate?: () => void };

// Shared SVG node-and-edge roadmap (works on Android, iOS and Web). Side
// tracks branch off beside the checkpoint they start after, like roadmap.sh.
function RoadmapCanvas({ course, nodes, levels, layout, width, nextNodeId, onNavigate }: Canvas) {
  const t = useT();
  const byId = useMemo(() => new Map(nodes.map((n) => [n.id, n])), [nodes]);

  // Keep the map readable with ~200 nodes: draw links inside a level, links
  // across levels only between ready lessons, and none into checkpoints
  // (they close their level). Every prerequisite is listed on the node page.
  const edges = nodes.flatMap((n) =>
    n.strand === 'CK'
      ? []
      : n.prerequisiteIds
          .map((p) => byId.get(p)!)
          .filter((from) => from.level === n.level || (from.status !== 'coming_soon' && n.status !== 'coming_soon'))
          .map((from) => {
            const active = from.status === 'mastered' && unlocked(n);
            return { key: `${from.id}->${n.id}`, d: edgePath(layout.boxes[from.id], layout.boxes[n.id]), active, soon: n.status === 'coming_soon' };
          }),
  );
  // A dotted link from the checkpoint to the branch's bar, which holds its cards.
  for (const l of layout.lanes) {
    const a = layout.boxes[l.anchor];
    const y = a.y + a.h / 2;
    const active = byId.get(l.anchor)?.status === 'mastered';
    edges.push({ key: `${l.anchor}->branch`, d: `M${a.x + a.w},${y} H${l.bar.x}`, active, soon: true });
    edges.push({ key: `${l.anchor}->bar`, d: `M${l.bar.x},${l.bar.y1} V${l.bar.y2}`, active, soon: false });
  }
  const nextLevel = (level: string) => levels[levels.findIndex((l) => l.level === level) + 1]?.level;

  return (
    <View style={{ width, height: layout.height }}>
      <Svg width={width} height={layout.height} style={StyleSheet.absoluteFill} pointerEvents="none">
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
        <View key={h.level} pointerEvents="none" style={[styles.header, { top: h.y }]}>
          <View style={styles.levelPill}>
            <Text size={12} weight="semibold" color={colors.white}>
              {h.level}
            </Text>
          </View>
          <Heading size={22}>{h.title}</Heading>
          <Eyebrow style={{ marginLeft: 'auto' }}>{t('course.levelMastered', { done: h.mastered, total: h.total })}</Eyebrow>
        </View>
      ))}
      {nodes.map((n) => (
        <NodeCard key={n.id} course={course} node={n} box={layout.boxes[n.id]} next={n.id === nextNodeId} onNavigate={onNavigate} />
      ))}
      {layout.lanes.map((l) => (
        <View key={l.anchor} pointerEvents="box-none" style={StyleSheet.absoluteFill}>
          <BranchHead lane={l} nextLevel={nextLevel(l.level)} />
          {l.cards.map((c) => (
            <TrackCard key={c.track.slug} track={c.track} box={c.box} level={l.level} onNavigate={onNavigate} />
          ))}
        </View>
      ))}
    </View>
  );
}

// The canvas drawn at `scale`, from its top-left corner.
function Scaled({ width, height, scale, children }: { width: number; height: number; scale: number; children: ReactNode }) {
  return (
    // overflow: clip on web, so focusing a card can't scroll the clipped box.
    <View style={[{ width: width * scale, height: height * scale, overflow: 'hidden' }, Platform.OS === 'web' && ({ overflow: 'clip' } as object)]}>
      <View style={{ width, height, transform: [{ scale }], transformOrigin: 'top left' }}>{children}</View>
    </View>
  );
}

const ZOOMS = [0.5, 0.75, 1, 1.25, 1.5];
type Point = { x: number; y: number };

// Full-screen magnifier: the map at a readable size, scrollable both ways,
// opened on the spot the learner tapped.
function ZoomView({ canvas, focus, onClose }: { canvas: Omit<Canvas, 'onNavigate'>; focus: Point; onClose: () => void }) {
  const t = useT();
  const insets = useSafeAreaInsets();
  const [zi, setZi] = useState(2);
  const z = ZOOMS[zi];
  const across = useRef<ScrollView>(null);
  const down = useRef<ScrollView>(null);
  // What's on screen, and the map point to keep in the middle of it.
  const view = useRef({ w: 0, h: 0, x: 0, y: 0 });
  const centre = useRef(focus);
  const pad = 24;

  useEffect(() => {
    const id = setTimeout(() => {
      const { w, h } = view.current;
      if (!w) return;
      across.current?.scrollTo({ x: Math.max(0, centre.current.x * z + pad - w / 2), animated: false });
      down.current?.scrollTo({ y: Math.max(0, centre.current.y * z + pad - h / 2), animated: false });
    }, 0);
    return () => clearTimeout(id);
  }, [z]);

  const zoom = (step: number) => {
    const { w, h, x, y } = view.current;
    centre.current = { x: (x + w / 2 - pad) / z, y: (y + h / 2 - pad) / z };
    setZi((i) => Math.min(ZOOMS.length - 1, Math.max(0, i + step)));
  };

  return (
    <Modal visible animationType="fade" onRequestClose={onClose}>
      <View style={[styles.zoomRoot, { paddingTop: insets.top }]}>
        <View style={styles.zoomBar}>
          <Text weight="medium" size={16} style={{ flex: 1 }} numberOfLines={1}>
            {t('course.roadmap')}
          </Text>
          <IconButton icon="zoomOut" label={t('course.zoomOut')} variant="soft" size={36} onPress={zi > 0 ? () => zoom(-1) : undefined} />
          <Text size={13} tabular style={{ width: 44, textAlign: 'center' }}>
            {Math.round(z * 100)}%
          </Text>
          <IconButton icon="zoomIn" label={t('course.zoomIn')} variant="soft" size={36} onPress={zi < ZOOMS.length - 1 ? () => zoom(1) : undefined} />
          <IconButton icon="close" label={t('common.close')} size={36} onPress={onClose} />
        </View>
        <ScrollView
          ref={down}
          style={{ flex: 1 }}
          scrollEventThrottle={32}
          onScroll={(e) => (view.current.y = e.nativeEvent.contentOffset.y)}
          onLayout={(e) => {
            const first = !view.current.w;
            view.current.w = e.nativeEvent.layout.width;
            view.current.h = e.nativeEvent.layout.height;
            if (first) {
              across.current?.scrollTo({ x: Math.max(0, centre.current.x * z + pad - view.current.w / 2), animated: false });
              down.current?.scrollTo({ y: Math.max(0, centre.current.y * z + pad - view.current.h / 2), animated: false });
            }
          }}
        >
          <ScrollView ref={across} horizontal scrollEventThrottle={32} onScroll={(e) => (view.current.x = e.nativeEvent.contentOffset.x)}>
            <View style={{ padding: pad, paddingBottom: pad + insets.bottom }}>
              <Scaled width={canvas.width} height={canvas.layout.height} scale={z}>
                <RoadmapCanvas {...canvas} onNavigate={onClose} />
              </Scaled>
            </View>
          </ScrollView>
        </ScrollView>
      </View>
    </Modal>
  );
}

// Phones get the same map as desktop, shrunk to fit like roadmap.sh's; a tap
// anywhere on it opens the magnifier there.
const DESIGN_W = 640;

export function RoadmapGraph({
  course,
  nodes,
  levels,
  width,
  wide,
  nextNodeId,
  branches = [],
}: {
  course: string;
  nodes: RoadmapNode[];
  levels: LevelSummary[];
  width: number;
  wide: boolean;
  nextNodeId: string | null;
  branches?: Branch[];
}) {
  const t = useT();
  const designW = wide ? width : Math.max(width, DESIGN_W);
  const layout = useMemo(() => layoutRoadmap(nodes, levels, designW, branches), [nodes, levels, designW, branches]);
  const [focus, setFocus] = useState<Point | null>(null);
  const frame = useRef<View>(null);
  const canvas = { course, nodes, levels, layout, width: designW, nextNodeId };
  if (wide) return <RoadmapCanvas {...canvas} />;

  const scale = width / designW;
  const next = nextNodeId ? layout.boxes[nextNodeId] : undefined;
  const openAt = (pageX: number, pageY: number) =>
    frame.current?.measureInWindow((x, y) => setFocus({ x: (pageX - x) / scale, y: (pageY - y) / scale }));
  return (
    <View style={{ gap: 10 }}>
      <View style={styles.zoomHint}>
        <Text size={13} color={colors.muted} style={{ flex: 1 }}>
          {t('course.zoomHint')}
        </Text>
        <Pressable
          onPress={() => setFocus(next ? { x: next.x + next.w / 2, y: next.y + next.h / 2 } : { x: designW / 2, y: 0 })}
          accessibilityRole="button"
          style={styles.zoomButton}
        >
          <Icon name="zoomIn" size={15} color={colors.white} />
          <Text size={13} weight="medium" color={colors.white}>
            {t('course.zoomOpen')}
          </Text>
        </Pressable>
      </View>
      <Pressable ref={frame} onPress={(e) => openAt(e.nativeEvent.pageX, e.nativeEvent.pageY)} accessibilityLabel={t('course.zoomOpen')}>
        <Scaled width={designW} height={layout.height} scale={scale}>
          <RoadmapCanvas {...canvas} />
        </Scaled>
      </Pressable>
      {focus ? <ZoomView canvas={canvas} focus={focus} onClose={() => setFocus(null)} /> : null}
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
  icons: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  check: { width: 18, height: 18, borderRadius: 9, backgroundColor: colors.emerald, alignItems: 'center', justifyContent: 'center' },
  pulse: { position: 'absolute', right: -10, top: -10 },
  header: { position: 'absolute', left: 0, right: 0, height: 40, flexDirection: 'row', alignItems: 'center', gap: 10 },
  levelPill: { backgroundColor: colors.ink, borderRadius: radius.pill, paddingHorizontal: 10, paddingVertical: 4 },
  track: { backgroundColor: colors.accentSoft, borderColor: colors.accent, borderWidth: 1 },
  branchHead: { position: 'absolute', gap: 3, justifyContent: 'flex-end', paddingBottom: 10 },
  zoomHint: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  zoomButton: { flexDirection: 'row', alignItems: 'center', gap: 6, height: 34, paddingHorizontal: 14, borderRadius: radius.pill, backgroundColor: colors.ink },
  zoomRoot: { flex: 1, backgroundColor: colors.canvas },
  zoomBar: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 16, paddingVertical: 10, borderBottomWidth: 1, borderBottomColor: colors.line, backgroundColor: colors.card },
});
