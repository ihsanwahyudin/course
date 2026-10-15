import type { Course, Level, LevelSummary, RoadmapNode } from '@/lib/api';

export type Box = { x: number; y: number; w: number; h: number };
export type LevelHeader = { level: Level; title: string; y: number; mastered: number; total: number };
export type GraphLayout = { boxes: Record<string, Box>; headers: LevelHeader[]; height: number; nodeW: number };

const HEADER_H = 56;
const NODE_H = 96;
const GAP_Y = 44;
const LEVEL_GAP = 28;

// Layered layout: one band per CEFR level; within a band, rows follow the
// longest same-level prerequisite chain, wrapped to the available width and
// ordered by the average x of each node's prerequisites to limit crossings.
export function layoutRoadmap(nodes: RoadmapNode[], levels: LevelSummary[], width: number, wide: boolean): GraphLayout {
  const gapX = wide ? 20 : 12;
  const nodeW = wide ? 196 : Math.max(136, (width - gapX) / 2);
  const perRow = Math.max(1, Math.floor((width + gapX) / (nodeW + gapX)));
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const boxes: Record<string, Box> = {};
  const headers: LevelHeader[] = [];
  let y = 0;

  for (const lvl of levels) {
    const inLevel = nodes.filter((n) => n.level === lvl.level);
    if (inLevel.length === 0) continue;
    headers.push({ level: lvl.level, title: lvl.title, y, mastered: lvl.mastered, total: lvl.total });
    y += HEADER_H;

    const depth = new Map<string, number>();
    for (const n of inLevel) {
      // nodes arrive topologically sorted, so same-level prereqs are known
      const d = n.prerequisiteIds.filter((p) => byId.get(p)?.level === lvl.level).map((p) => (depth.get(p) ?? 0) + 1);
      depth.set(n.id, Math.max(0, ...d));
    }
    const maxDepth = Math.max(...depth.values());
    for (let d = 0; d <= maxDepth; d++) {
      const row = inLevel.filter((n) => depth.get(n.id) === d);
      const anchor = (n: RoadmapNode) => {
        const xs = n.prerequisiteIds.map((p) => boxes[p]).filter(Boolean).map((b) => b.x);
        return xs.length ? xs.reduce((a, b) => a + b, 0) / xs.length : Number.MAX_SAFE_INTEGER / 2;
      };
      row.sort((a, b) => anchor(a) - anchor(b));
      for (let i = 0; i < row.length; i += perRow) {
        const chunk = row.slice(i, i + perRow);
        const rowW = chunk.length * nodeW + (chunk.length - 1) * gapX;
        let x = (width - rowW) / 2;
        for (const n of chunk) {
          boxes[n.id] = { x, y, w: nodeW, h: NODE_H };
          x += nodeW + gapX;
        }
        y += NODE_H + GAP_Y;
      }
    }
    y += LEVEL_GAP - GAP_Y + 12;
  }
  return { boxes, headers, height: y, nodeW };
}

// Smooth vertical connector from the bottom of `a` to the top of `b`.
export function edgePath(a: Box, b: Box) {
  const x1 = a.x + a.w / 2;
  const y1 = a.y + a.h;
  const x2 = b.x + b.w / 2;
  const y2 = b.y;
  const dy = Math.max(24, (y2 - y1) / 2);
  return `M${x1},${y1} C${x1},${y1 + dy} ${x2},${y2 - dy} ${x2},${y2}`;
}

// A side track (TOEFL, IELTS, …) that branches off this roadmap after a
// level, like roadmap.sh's "relevant tracks", drawn in a lane to the right.
export type Branch = { anchor: string; level: Level; tracks: Course[] };
export type BranchLayout = { anchor: string; level: Level; head: Box; cards: { track: Course; box: Box }[] };

// Tracks recommended after a level of this course hang off that level's
// checkpoint (or its last lesson); the rest have no place on the map.
export function branchesFor(course: string, nodes: RoadmapNode[], tracks: Course[]): { branches: Branch[]; rest: Course[] } {
  const branches: Branch[] = [];
  const rest: Course[] = [];
  for (const tr of tracks) {
    const after = tr.recommendedAfter;
    const inLevel = after && after.course === course ? nodes.filter((n) => n.level === after.level) : [];
    const anchor = inLevel.find((n) => n.strand === 'CK') ?? inLevel[inLevel.length - 1];
    if (!anchor) {
      rest.push(tr);
      continue;
    }
    const b = branches.find((x) => x.anchor === anchor.id);
    if (b) b.tracks.push(tr);
    else branches.push({ anchor: anchor.id, level: anchor.level, tracks: [tr] });
  }
  return { branches, rest };
}

export const LANE_GAP = 120;
const BRANCH_HEAD_H = 92;
const BRANCH_CARD_H = 84;
const BRANCH_GAP_Y = 12;

// Stacks each branch's cards in the lane, centred on its anchor's row.
export function layoutBranches(branches: Branch[], boxes: Record<string, Box>, laneX: number, laneW: number): { lanes: BranchLayout[]; height: number } {
  let floor = 0;
  let height = 0;
  const lanes = branches.map((b) => {
    const a = boxes[b.anchor];
    const total = BRANCH_HEAD_H + b.tracks.length * (BRANCH_CARD_H + BRANCH_GAP_Y) - BRANCH_GAP_Y;
    let y = Math.max(floor, a.y + a.h / 2 - total / 2);
    const head = { x: laneX, y, w: laneW, h: BRANCH_HEAD_H };
    y += BRANCH_HEAD_H;
    const cards = b.tracks.map((track) => {
      const box = { x: laneX, y, w: laneW, h: BRANCH_CARD_H };
      y += BRANCH_CARD_H + BRANCH_GAP_Y;
      return { track, box };
    });
    floor = y + 32;
    height = Math.max(height, y);
    return { anchor: b.anchor, level: b.level, head, cards };
  });
  return { lanes, height };
}

// Smooth sideways connector from the right of `a` to the left of `b`.
export function branchPath(a: Box, b: Box) {
  const x1 = a.x + a.w;
  const y1 = a.y + a.h / 2;
  const x2 = b.x;
  const y2 = b.y + b.h / 2;
  const dx = (x2 - x1) / 2;
  return `M${x1},${y1} C${x1 + dx},${y1} ${x2 - dx},${y2} ${x2},${y2}`;
}
