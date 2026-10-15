import type { Course, Level, LevelSummary, RoadmapNode } from '@/lib/api';

export type Box = { x: number; y: number; w: number; h: number };
export type LevelHeader = { level: Level; title: string; y: number; mastered: number; total: number };
// A side track (TOEFL, IELTS, …) that branches off this roadmap after a
// level, drawn beside that level's checkpoint like roadmap.sh's side panels.
export type Branch = { anchor: string; level: Level; tracks: Course[] };
export type BranchLayout = { anchor: string; level: Level; head: Box; cards: { track: Course; box: Box }[]; bar: { x: number; y1: number; y2: number } };
export type GraphLayout = { boxes: Record<string, Box>; headers: LevelHeader[]; lanes: BranchLayout[]; height: number; nodeW: number };

const HEADER_H = 56;
const NODE_H = 96;
const GAP_Y = 44;
const LEVEL_GAP = 28;
const BRANCH_HEAD_H = 84;
const BRANCH_GAP_Y = 10;

// Layered layout: one band per CEFR level; within a band, rows follow the
// longest same-level prerequisite chain, wrapped to the available width and
// ordered by the average x of each node's prerequisites to limit crossings.
// A row holding a branch anchor moves left to make room for the branch's
// cards on the right, and grows tall enough to hold them.
// Phones draw the same layout, scaled down (see RoadmapGraph).
export function layoutRoadmap(nodes: RoadmapNode[], levels: LevelSummary[], width: number, branches: Branch[] = []): GraphLayout {
  const gapX = 20;
  const nodeW = 196;
  const perRow = Math.max(1, Math.floor((width + gapX) / (nodeW + gapX)));
  const laneW = 260;
  const laneGap = 64;
  const leftW = width - laneW - laneGap;
  const perLeftRow = Math.max(1, Math.floor((leftW + gapX) / (nodeW + gapX)));
  const cardH = 72;
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const branchOf = new Map(branches.map((b) => [b.anchor, b]));
  const boxes: Record<string, Box> = {};
  const headers: LevelHeader[] = [];
  const lanes: BranchLayout[] = [];
  let y = 0;

  const place = (chunk: RoadmapNode[], from: number, span: number, top: number) => {
    const rowW = chunk.length * nodeW + (chunk.length - 1) * gapX;
    let x = from + (span - rowW) / 2;
    for (const n of chunk) {
      boxes[n.id] = { x, y: top, w: nodeW, h: NODE_H };
      x += nodeW + gapX;
    }
  };

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
      const branch = row.map((n) => branchOf.get(n.id)).find(Boolean);
      if (!branch) {
        for (let i = 0; i < row.length; i += perRow) {
          place(row.slice(i, i + perRow), 0, width, y);
          y += NODE_H + GAP_Y;
        }
        continue;
      }
      // Branch row: the cards stack on the right and the row sits level
      // with their middle, on the left.
      const n = branch.tracks.length;
      const cardsH = n * cardH + (n - 1) * BRANCH_GAP_Y;
      const laneX = width - laneW;
      const cardsTop = y + BRANCH_HEAD_H;
      const cards = branch.tracks.map((track, i) => ({ track, box: { x: laneX, y: cardsTop + i * (cardH + BRANCH_GAP_Y), w: laneW, h: cardH } }));
      lanes.push({
        anchor: branch.anchor,
        level: branch.level,
        head: { x: laneX, y, w: laneW, h: BRANCH_HEAD_H },
        cards,
        bar: { x: laneX - 14, y1: cardsTop, y2: cardsTop + cardsH },
      });
      const chunks = [];
      for (let i = 0; i < row.length; i += perLeftRow) chunks.push(row.slice(i, i + perLeftRow));
      const rowsH = chunks.length * NODE_H + (chunks.length - 1) * GAP_Y;
      let top = cardsTop + cardsH / 2 - rowsH / 2;
      for (const c of chunks) {
        place(c, 0, leftW, top);
        top += NODE_H + GAP_Y;
      }
      y = Math.max(cardsTop + cardsH, top - GAP_Y) + GAP_Y;
    }
    y += LEVEL_GAP - GAP_Y + 12;
  }
  return { boxes, headers, lanes, height: y, nodeW };
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
