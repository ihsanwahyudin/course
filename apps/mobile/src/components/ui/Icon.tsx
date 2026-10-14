import Svg, { Circle, Path } from 'react-native-svg';

import { colors } from '@/theme/tokens';

// Small stroke icon set (24px grid, 1.75 stroke) in the spirit of the site's icons.
const paths = {
  arrowUpRight: ['M7 17 17 7', 'M8 7h9v9'],
  arrowLeft: ['M19 12H5', 'm12 19-7-7 7-7'],
  arrowRight: ['M5 12h14', 'm12 5 7 7-7 7'],
  check: ['M20 6 9 17l-5-5'],
  lock: ['M7 11V7a5 5 0 0 1 10 0v4', 'M5 11h14v10H5z'],
  send: ['m22 2-7 20-4-9-9-4Z', 'M22 2 11 13'],
  route: ['M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15'],
  home: ['m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z'],
  map: ['M9 4 3 6v14l6-2 6 2 6-2V4l-6 2z', 'M9 4v14', 'M15 6v14'],
  chart: ['M3 3v18h18', 'M7 15l4-4 3 3 5-6'],
  refresh: ['M21 12a9 9 0 1 1-3-6.7L21 8', 'M21 3v5h-5'],
  close: ['M18 6 6 18', 'm6 6 12 12'],
  chevronRight: ['m9 18 6-6-6-6'],
  sparkle: ['M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z'],
  list: ['M8 6h13', 'M8 12h13', 'M8 18h13', 'M3 6h.01', 'M3 12h.01', 'M3 18h.01'],
  graph: ['M6 3v6', 'M18 15v6', 'M6 9a3 3 0 1 0 0 0', 'M8.5 10.5l7 3'],
  target: [],
  user: ['M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2', 'M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z'],
  book: ['M4 19.5A2.5 2.5 0 0 1 6.5 17H20', 'M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z'],
  play: ['M7 4v16l13-8z'],
  logout: ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', 'm16 17 5-5-5-5', 'M21 12H9'],
  flame: ['M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.07-2.14-.22-4.05 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.15.43-2.29 1-3a2.5 2.5 0 0 0 2.5 2.5z'],
  mail: ['M4 4h16v16H4z', 'm22 6-10 7L2 6'],
  mic: ['M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z', 'M19 10v2a7 7 0 0 1-14 0v-2', 'M12 19v3'],
  volume: ['M11 5 6 9H2v6h4l5 4V5Z', 'M15.54 8.46a5 5 0 0 1 0 7.07', 'M19.07 4.93a10 10 0 0 1 0 14.14'],
  bulb: ['M9 18h6', 'M10 22h4', 'M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14'],
  flag: ['M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z', 'M4 22v-7'],
  chat: ['M7.9 20A9 9 0 1 0 4 16.1L2 22Z'],
  layers: ['m12 2 10 5-10 5L2 7z', 'm2 17 10 5 10-5', 'm2 12 10 5 10-5'],
  brain: [
    'M12 5a3 3 0 1 0-5.997.125 4 4 0 0 0-2.526 5.77 4 4 0 0 0 .556 6.588A4 4 0 1 0 12 18Z',
    'M12 5a3 3 0 1 1 5.997.125 4 4 0 0 1 2.526 5.77 4 4 0 0 1-.556 6.588A4 4 0 1 1 12 18Z',
    'M12 5v13',
  ],
  edit: ['M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z'],
  trash: ['M3 6h18', 'M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6', 'M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2'],
  panel: ['M3 3h18v18H3z', 'M15 3v18'],
  arrowUp: ['m5 12 7-7 7 7', 'M12 19V5'],
  stop: ['M6 6h12v12H6z'],
  undo: ['M3 7v6h6', 'M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13'],
} as const;

export type IconName = keyof typeof paths;

export function Icon({ name, size = 18, color = colors.ink, strokeWidth = 1.75 }: { name: IconName; size?: number; color?: string; strokeWidth?: number }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round">
      {name === 'graph' ? (
        <>
          <Circle cx="6" cy="6" r="3" />
          <Circle cx="18" cy="18" r="3" />
          <Path d="M8.5 7.5l7 8" />
        </>
      ) : name === 'target' ? (
        <>
          <Circle cx="12" cy="12" r="9" />
          <Circle cx="12" cy="12" r="5" />
          <Circle cx="12" cy="12" r="1" />
        </>
      ) : (
        paths[name].map((d) => <Path key={d} d={d} />)
      )}
    </Svg>
  );
}
