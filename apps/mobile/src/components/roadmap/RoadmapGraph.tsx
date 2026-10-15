import { useRouter } from "expo-router";
import { useMemo, useRef, useState } from "react";
import {
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
} from "react-native";
import Svg, { Path } from "react-native-svg";

import { GoalBadge } from "@/components/ui/Bits";
import { PulseDot } from "@/components/ui/Decor";
import { Icon } from "@/components/ui/Icon";
import { Eyebrow, Heading, Text } from "@/components/ui/Text";
import type { Course, LevelSummary, RoadmapNode } from "@/lib/api";
import { useT } from "@/i18n";
import { colors, radius, shadow } from "@/theme/tokens";
import {
  branchPath,
  edgePath,
  LANE_GAP,
  layoutBranches,
  layoutRoadmap,
  type Box,
  type Branch,
  type BranchLayout,
} from "./layout";

const unlocked = (n: RoadmapNode) =>
  n.status === "available" ||
  n.status === "in_progress" ||
  n.status === "mastered";

function NodeCard({
  course,
  node,
  box,
  next,
}: {
  course: string;
  node: RoadmapNode;
  box: Box;
  next: boolean;
}) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  const t = useT();
  const s = node.status;
  const bg =
    s === "mastered"
      ? colors.mint
      : s === "in_progress"
        ? colors.lavender
        : s === "coming_soon"
          ? "rgba(255,255,255,0.55)"
          : colors.card;
  return (
    <Pressable
      onPress={() => router.push(`/courses/${course}/${node.id}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={`${node.title}, ${node.level}, ${t(`status.${s}`)}${node.recommended ? `, ${t("onboarding.recommended")}` : ""}`}
      style={[
        styles.node,
        {
          left: box.x,
          top: box.y,
          width: box.w,
          height: box.h,
          backgroundColor: bg,
          borderColor: s === "available" || next ? colors.accent : colors.line,
          borderWidth: s === "available" || next ? 1.5 : 1,
          borderStyle: s === "coming_soon" ? "dashed" : "solid",
          opacity: s === "locked" ? 0.6 : 1,
          boxShadow:
            s === "coming_soon"
              ? "none"
              : hovered
                ? shadow.cardHover
                : shadow.card,
          transform: [{ translateY: hovered ? -2 : 0 }],
        },
        Platform.OS === "web" &&
          ({
            transitionProperty: "box-shadow, transform",
            transitionDuration: "400ms",
          } as object),
      ]}
    >
      <View style={styles.nodeTop}>
        <Eyebrow
          color={
            s === "mastered"
              ? colors.emeraldDeep
              : s === "available" || s === "in_progress"
                ? colors.accent
                : colors.muted
          }
        >
          {node.level} ·{" "}
          {s === "coming_soon" ? t("status.soon") : t(`status.${s}`)}
        </Eyebrow>
        <View style={styles.icons}>
          {node.recommended && s !== "mastered" ? <GoalBadge compact /> : null}
          {s === "mastered" ? (
            <View style={styles.check}>
              <Icon
                name="check"
                size={12}
                color={colors.white}
                strokeWidth={2.5}
              />
            </View>
          ) : s === "locked" ? (
            <Icon name="lock" size={14} color={colors.muted} />
          ) : null}
        </View>
      </View>
      <Text
        size={14}
        weight="medium"
        tight
        numberOfLines={2}
        color={s === "coming_soon" ? colors.muted : colors.ink}
      >
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

function TrackCard({
  track,
  box,
  level,
}: {
  track: Course;
  box: Box;
  level: string;
}) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  const t = useT();
  const soon = track.status === "coming_soon";
  return (
    <Pressable
      onPress={() => router.push(`/courses/${track.slug}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      accessibilityLabel={`${track.title}, ${t("course.branchAfter", { level })}${soon ? `, ${t("common.comingSoon")}` : ""}`}
      style={[
        styles.node,
        styles.track,
        {
          left: box.x,
          top: box.y,
          width: box.w,
          height: box.h,
          borderStyle: soon ? "dashed" : "solid",
          boxShadow: hovered ? shadow.cardHover : "none",
          transform: [{ translateY: hovered ? -2 : 0 }],
        },
        Platform.OS === "web" &&
          ({
            transitionProperty: "box-shadow, transform",
            transitionDuration: "400ms",
          } as object),
      ]}
    >
      <View style={styles.nodeTop}>
        <Eyebrow color={colors.accent}>
          {track.category} ·{" "}
          {soon ? t("status.soon") : t("course.branchAfter", { level })}
        </Eyebrow>
        <Icon name="arrowUpRight" size={14} color={colors.accent} />
      </View>
      <Text size={15} weight="medium" tight numberOfLines={1}>
        {track.title}
      </Text>
    </Pressable>
  );
}

// Head of a branch lane: where it starts and the way back to the main road.
function BranchHead({
  lane,
  nextLevel,
  onBack,
}: {
  lane: BranchLayout;
  nextLevel?: string;
  onBack: () => void;
}) {
  const t = useT();
  return (
    <View
      style={[
        styles.branchHead,
        {
          left: lane.head.x,
          top: lane.head.y,
          width: lane.head.w,
          height: lane.head.h,
        },
      ]}
    >
      <Eyebrow>{t("course.branchAfter", { level: lane.level })}</Eyebrow>
      <Heading size={18}>{t("course.branchTitle")}</Heading>
      <Pressable
        onPress={onBack}
        accessibilityRole="button"
        hitSlop={8}
        style={styles.backLink}
      >
        <Icon name="arrowLeft" size={13} color={colors.muted} />
        <Text size={12} color={colors.muted} numberOfLines={1}>
          {nextLevel
            ? t("course.branchBack", { level: nextLevel })
            : t("course.branchBackPlain")}
        </Text>
      </Pressable>
    </View>
  );
}

// Shared SVG node-and-edge roadmap (works on Android, iOS and Web). Side
// tracks branch off to the right in a lane the learner scrolls to.
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
  const scroller = useRef<ScrollView>(null);
  const layout = useMemo(
    () => layoutRoadmap(nodes, levels, width, wide),
    [nodes, levels, width, wide],
  );
  const byId = useMemo(() => new Map(nodes.map((n) => [n.id, n])), [nodes]);
  const laneW = wide ? 248 : 208;
  const laneX = width + LANE_GAP;
  const lane = useMemo(
    () =>
      branches.length
        ? layoutBranches(branches, layout.boxes, laneX, laneW)
        : null,
    [branches, layout, laneX, laneW],
  );
  const canvasW = lane ? laneX + laneW + 4 : width;
  const height = Math.max(layout.height, lane?.height ?? 0);

  // Keep the map readable with ~200 nodes: draw links inside a level, links
  // across levels only between ready lessons, and none into checkpoints
  // (they close their level). Every prerequisite is listed on the node page.
  const edges = nodes.flatMap((n) =>
    n.strand === "CK"
      ? []
      : n.prerequisiteIds
          .map((p) => byId.get(p)!)
          .filter(
            (from) =>
              from.level === n.level ||
              (from.status !== "coming_soon" && n.status !== "coming_soon"),
          )
          .map((from) => {
            const active = from.status === "mastered" && unlocked(n);
            return {
              key: `${from.id}->${n.id}`,
              d: edgePath(layout.boxes[from.id], layout.boxes[n.id]),
              active,
              soon: n.status === "coming_soon",
            };
          }),
  );
  for (const l of lane?.lanes ?? []) {
    const active = byId.get(l.anchor)?.status === "mastered";
    for (const c of l.cards) {
      edges.push({
        key: `${l.anchor}->${c.track.slug}`,
        d: branchPath(layout.boxes[l.anchor], c.box),
        active,
        soon: !active,
      });
    }
  }
  const nextLevel = (level: string) =>
    levels[levels.findIndex((l) => l.level === level) + 1]?.level;

  const canvas = (
    <View style={{ width: canvasW, height }}>
      <Svg width={canvasW} height={height} style={StyleSheet.absoluteFill}>
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
              strokeDasharray={e.soon ? "4 5" : undefined}
            />
          ))}
      </Svg>
      {layout.headers.map((h) => (
        <View key={h.level} style={[styles.header, { top: h.y, width }]}>
          <View style={styles.levelPill}>
            <Text size={12} weight="semibold" color={colors.white}>
              {h.level}
            </Text>
          </View>
          <Heading size={wide ? 22 : 18}>{h.title}</Heading>
          <Eyebrow style={{ marginLeft: "auto" }}>
            {t("course.levelMastered", { done: h.mastered, total: h.total })}
          </Eyebrow>
        </View>
      ))}
      {nodes.map((n) => (
        <NodeCard
          key={n.id}
          course={course}
          node={n}
          box={layout.boxes[n.id]}
          next={n.id === nextNodeId}
        />
      ))}
      {lane?.lanes.map((l) => {
        const a = layout.boxes[l.anchor];
        return (
          <View
            key={l.anchor}
            pointerEvents="box-none"
            style={StyleSheet.absoluteFill}
          >
            {/* Signpost at the fork: the lane starts off-screen to the right. */}
            <Pressable
              onPress={() =>
                scroller.current?.scrollTo({
                  x: Math.max(0, canvasW - width),
                  animated: true,
                })
              }
              accessibilityRole="button"
              style={[
                styles.fork,
                // Beside the checkpoint when there's room, otherwise just below it.
                { right: canvasW - width + 4, top: width - (a.x + a.w) >= 150 ? a.y + a.h / 2 - 17 : a.y + a.h + 8 },
              ]}
            >
              <Text size={13} weight="medium" color={colors.white}>
                {t("course.branchFork")}
              </Text>
              <Icon name="arrowRight" size={14} color={colors.white} />
            </Pressable>
            <BranchHead
              lane={l}
              nextLevel={nextLevel(l.level)}
              onBack={() =>
                scroller.current?.scrollTo({ x: 0, animated: true })
              }
            />
            {l.cards.map((c) => (
              <TrackCard
                key={c.track.slug}
                track={c.track}
                box={c.box}
                level={l.level}
              />
            ))}
          </View>
        );
      })}
    </View>
  );

  if (!lane) return canvas;
  return (
    <ScrollView
      ref={scroller}
      horizontal
      style={{ width }}
      contentContainerStyle={{ width: canvasW }}
      showsHorizontalScrollIndicator
    >
      {canvas}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  node: {
    position: "absolute",
    borderRadius: radius.xxl + 4,
    padding: 14,
    justifyContent: "space-between",
  },
  nodeTop: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 6,
  },
  icons: { flexDirection: "row", alignItems: "center", gap: 4 },
  check: {
    width: 18,
    height: 18,
    borderRadius: 9,
    backgroundColor: colors.emerald,
    alignItems: "center",
    justifyContent: "center",
  },
  pulse: { position: "absolute", right: -10, top: -10 },
  header: {
    position: "absolute",
    left: 0,
    height: 40,
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
  },
  levelPill: {
    backgroundColor: colors.ink,
    borderRadius: radius.pill,
    paddingHorizontal: 10,
    paddingVertical: 4,
  },
  track: {
    backgroundColor: colors.accentSoft,
    borderColor: colors.accent,
    borderWidth: 1,
  },
  branchHead: { position: "absolute", gap: 4, justifyContent: "center" },
  backLink: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    marginTop: 2,
    alignSelf: "flex-start",
  },
  fork: {
    position: "absolute",
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    height: 34,
    paddingHorizontal: 14,
    borderRadius: radius.pill,
    backgroundColor: colors.accent,
  },
});
