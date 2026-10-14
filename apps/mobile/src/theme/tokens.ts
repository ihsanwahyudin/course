// Design tokens taken from ihsan.my.id so FluentPath shares its look:
// warm dotted canvas, white bento cards, Geist + Geist Mono and a violet
// accent. Kept deliberately calm for long study sessions: white surfaces,
// one accent, and pale tints only where they carry meaning (success, review).
// The violet → magenta → orange gradient survives only in the logo mark.
// Light theme only, like the site.

export const colors = {
  canvas: '#f3f2ee',
  card: '#ffffff',
  ink: '#111113',
  muted: '#6e6e76',
  line: 'rgba(17,17,19,0.08)',
  lineStrong: 'rgba(17,17,19,0.14)',
  dot: 'rgba(17,17,19,0.05)',
  inkSoft: 'rgba(17,17,19,0.06)',
  accent: '#5b4cf5',
  accentSoft: '#f0efff',
  magenta: '#c04cf5',
  orange: '#f5844c',
  lavender: '#e6e2ff',
  peach: '#fdf0e8',
  mint: '#e4f5ec',
  sky: '#eaf2fc',
  emerald: '#00bb7f',
  emeraldDeep: '#009767',
  danger: '#d6453d',
  // Hints: a pale amber so they read as help, not as an error.
  amber: '#9a6400',
  amberSoft: '#fdf6e3',
  white: '#ffffff',
} as const;

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
  dot: '0px 2px 6px rgba(91,76,245,0.25)',
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
