import type React from 'react';
import { StyleSheet, Text as RNText, type TextProps, type TextStyle } from 'react-native';

import { colors, fonts } from '@/theme/tokens';

type Weight = 'regular' | 'medium' | 'semibold';

type Props = TextProps & {
  size?: number;
  weight?: Weight;
  color?: string;
  tight?: boolean;
  tabular?: boolean;
};

// Body text in Geist. `tight` applies the site's tracking-tight (-0.025em).
export function Text({ size = 15, weight = 'regular', color = colors.ink, tight, tabular, style, ...rest }: Props) {
  const s: TextStyle = {
    fontFamily: fonts[weight],
    fontSize: size,
    lineHeight: Math.round(size * 1.5),
    color,
    letterSpacing: tight ? -0.025 * size : 0,
    fontVariant: tabular ? ['tabular-nums'] : undefined,
  };
  return <RNText {...rest} style={[s, style]} />;
}

// `.heading`: semibold, letter-spacing -0.045em, line-height ~0.95.
export function Heading({ size = 32, color = colors.ink, tabular, style, ...rest }: Omit<Props, 'weight' | 'tight'>) {
  return (
    <RNText
      accessibilityRole="header"
      {...rest}
      style={[
        {
          fontFamily: fonts.semibold,
          fontSize: size,
          lineHeight: Math.round(size * 1.02),
          letterSpacing: -0.045 * size,
          color,
          fontVariant: tabular ? ['tabular-nums'] : undefined,
        },
        style,
      ]}
    />
  );
}

// `.eyebrow`: Geist Mono, uppercase, 0.06em tracking, muted, ~11.5px.
export function Eyebrow({ color = colors.muted, style, children, ...rest }: Omit<Props, 'size' | 'weight'>) {
  return (
    <RNText {...rest} style={[styles.eyebrow, { color }, style]}>
      {children}
    </RNText>
  );
}

const styles = StyleSheet.create({
  eyebrow: {
    fontFamily: fonts.mono,
    fontSize: 11.5,
    lineHeight: 16,
    letterSpacing: 0.69,
    textTransform: 'uppercase',
  },
});

// Inline run inside a Heading/Text that only changes colour (no extra
// heading semantics), e.g. the accent "%" or "+" after a big number.
export function Span({ color, children }: { color: string; children: React.ReactNode }) {
  return <RNText style={{ color }}>{children}</RNText>;
}
