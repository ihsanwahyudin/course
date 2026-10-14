import type { Level, LevelSummary, RoadmapNode } from '@/lib/api';

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
