// Design tokens taken from ihsan.my.id so FluentPath shares its look:
// warm dotted canvas, white bento cards, Geist + Geist Mono, violet accent
// and a violet → magenta → orange brand gradient. Light theme only, like the site.

export const colors = {
  canvas: '#f3f2ee',
  card: '#ffffff',
  ink: '#111113',
  muted: '#6e6e76',
  line: 'rgba(17,17,19,0.08)',
  lineStrong: 'rgba(17,17,19,0.14)',
  dot: 'rgba(17,17,19,0.09)',
  inkSoft: 'rgba(17,17,19,0.06)',
  accent: '#5b4cf5',
  accentSoft: '#ecebff',
  magenta: '#c04cf5',
  orange: '#f5844c',
  lavender: '#ded8ff',
  peach: '#ffdcc7',
  mint: '#cdeedd',
  sky: '#d3e7ff',
  emerald: '#00bb7f',
  emeraldDeep: '#009767',
  danger: '#d6453d',
  white: '#ffffff',
} as const;

export const gradient = [colors.accent, colors.magenta, colors.orange] as const;
// Flowing headline gradient (accent → magenta → orange → magenta → accent).
export const gradientFlow = [colors.accent, colors.magenta, colors.orange, colors.magenta, colors.accent] as const;

export type Tone = 'default' | 'lavender' | 'peach' | 'mint' | 'sky' | 'ink';

export const toneBg: Record<Tone, string> = {
  default: colors.card,
  lavender: colors.lavender,
  peach: colors.peach,
  mint: colors.mint,
  sky: colors.sky,
  ink: colors.ink,
};

export const radius = {
  card: 28,
  xxl: 16,
  xl: 12,
  lg: 8,
  pill: 999,
} as const;

export const space = (n: number) => n * 4;

export const shadow = {
  card: '0px 1px 2px rgba(17,17,19,0.04), 0px 12px 32px -16px rgba(17,17,19,0.12)',
  cardHover: '0px 1px 2px rgba(17,17,19,0.04), 0px 24px 48px -20px rgba(17,17,19,0.22)',
  dot: '0px 4px 10px rgba(91,76,245,0.35)',
} as const;

export const fonts = {
  regular: 'Geist_400Regular',
  medium: 'Geist_500Medium',
  semibold: 'Geist_600SemiBold',
  mono: 'GeistMono_400Regular',
  monoMedium: 'GeistMono_500Medium',
} as const;

// ease-out-expo, the site's signature easing
export const easeOutExpo = [0.16, 1, 0.3, 1] as const;

export const layout = {
  maxWidth: 1152,
  gutter: 16,
  gutterWide: 24,
  gap: 12,
  gapWide: 16,
  wideBreakpoint: 768,
  navClearance: 96,
  navClearanceWide: 112,
} as const;
