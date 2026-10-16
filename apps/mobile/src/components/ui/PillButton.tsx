import { useState } from 'react';
import { ActivityIndicator, Platform, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';

import { colors, fonts, radius } from '@/theme/tokens';
import { Icon, type IconName } from './Icon';
import { Text } from './Text';

type Variant = 'primary' | 'secondary' | 'accent';

type Props = {
  label: string;
  onPress?: () => void;
  variant?: Variant;
  icon?: IconName;
  disabled?: boolean;
  loading?: boolean;
  size?: 'md' | 'sm';
  style?: StyleProp<ViewStyle>;
  accessibilityHint?: string;
};

// Site buttons: primary = ink pill that turns accent on hover (icon rotates
// 45°), secondary = white bordered pill, accent = violet pill that turns ink.
export function PillButton({ label, onPress, variant = 'primary', icon, disabled, loading, size = 'md', style, accessibilityHint }: Props) {
  const [hovered, setHovered] = useState(false);
  const inactive = disabled || loading;
  const bg =
    variant === 'secondary'
      ? hovered && !inactive ? colors.canvas : colors.card
      : variant === 'accent'
        ? hovered && !inactive ? colors.ink : colors.accent
        : hovered && !inactive ? colors.accent : colors.ink;
  const fg = variant === 'secondary' ? colors.ink : colors.white;
  return (
    <Pressable
      onPress={onPress}
      disabled={inactive}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityHint={accessibilityHint}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      style={({ pressed }) => [
        styles.base,
        size === 'sm' ? styles.sm : styles.md,
        { backgroundColor: bg, opacity: disabled ? 0.45 : 1 },
        variant === 'secondary' && styles.bordered,
        pressed && !inactive && { transform: [{ scale: 0.97 }] },
        Platform.OS === 'web' && ({ transitionProperty: 'background-color', transitionDuration: '150ms', cursor: inactive ? 'not-allowed' : 'pointer' } as object),
        style,
      ]}
    >
      {loading ? <ActivityIndicator size="small" color={fg} /> : null}
      <Text size={size === 'sm' ? 13 : 14} weight="medium" color={fg} style={{ fontFamily: fonts.medium, flexShrink: 1, textAlign: 'center' }}>
        {label}
      </Text>
      {icon && !loading ? (
        <View style={{ transform: [{ rotate: hovered && icon === 'arrowUpRight' ? '45deg' : '0deg' }] }}>
          <Icon name={icon} size={16} color={fg} />
        </View>
      ) : null}
    </Pressable>
  );
}

// Round icon-only button (e.g. the ink ↗ circle in a tile corner).
export function IconButton({ icon, onPress, label, variant = 'ink', size = 40 }: { icon: IconName; onPress?: () => void; label: string; variant?: 'ink' | 'soft' | 'accent'; size?: number }) {
  const [hovered, setHovered] = useState(false);
  const bg = variant === 'soft' ? (hovered ? colors.lineStrong : colors.inkSoft) : variant === 'accent' ? (hovered ? colors.ink : colors.accent) : hovered ? colors.accent : colors.ink;
  const fg = variant === 'soft' ? colors.ink : colors.white;
  return (
    <Pressable
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={6}
      style={({ pressed }) => [
        { width: size, height: size, borderRadius: radius.pill, backgroundColor: bg, alignItems: 'center', justifyContent: 'center' },
        pressed && { transform: [{ scale: 0.94 }] },
      ]}
    >
      <Icon name={icon} size={Math.round(size * 0.45)} color={fg} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: radius.pill,
    alignSelf: 'flex-start',
    // A long label (a lesson title) wraps instead of running off a phone screen.
    maxWidth: '100%',
  },
  md: { paddingHorizontal: 20, paddingVertical: 12, minHeight: 44 },
  sm: { paddingHorizontal: 14, paddingVertical: 8, minHeight: 36 },
  bordered: { borderWidth: 1, borderColor: colors.line },
});

// Decorative ink circle with ↗ for tiles that are themselves pressable
// (a nested button would be invalid on web).
export function ArrowBadge({ size = 36, color = colors.ink }: { size?: number; color?: string }) {
  return (
    <View accessible={false} style={{ width: size, height: size, borderRadius: radius.pill, backgroundColor: color, alignItems: 'center', justifyContent: 'center' }}>
      <Icon name="arrowUpRight" size={Math.round(size * 0.45)} color={colors.white} />
    </View>
  );
}
