import { createElement, useEffect, useRef, useState, type ReactNode } from 'react';
import { AccessibilityInfo, Animated, Easing, Platform, Pressable, StyleSheet, useWindowDimensions, View, type StyleProp, type ViewStyle } from 'react-native';

import { colors, easeOutExpo, radius, shadow, toneBg, type Tone } from '@/theme/tokens';

const AnimatedPressable = Animated.createAnimatedComponent(Pressable);

type Props = {
  tone?: Tone;
  children?: ReactNode;
  style?: StyleProp<ViewStyle>;
  padding?: number;
  onPress?: () => void;
  accessibilityLabel?: string;
  /** Background decoration rendered behind content (mesh blobs, dot pattern). */
  decoration?: ReactNode;
  /** Stagger index for the entrance animation; omit to show immediately. */
  reveal?: number;
  /** Pointer-following violet spotlight on hover (web). */
  spotlight?: boolean;
};

// Entrance like the site's hero cards: rise 48px, scale from .96, fade in,
// expo-out over 1.2s, staggered by 70ms.
function useReveal(index?: number) {
  const v = useState(() => new Animated.Value(index === undefined ? 1 : 0))[0];
  useEffect(() => {
    if (index === undefined) return;
    let cancelled = false;
    AccessibilityInfo.isReduceMotionEnabled()
      .catch(() => false)
      .then((reduce) => {
        if (cancelled) return;
        if (reduce) return v.setValue(1);
        Animated.timing(v, {
          toValue: 1,
          duration: 1200,
          delay: 60 + index * 70,
          easing: Easing.bezier(...easeOutExpo),
          useNativeDriver: Platform.OS !== 'web',
        }).start();
      });
    return () => {
      cancelled = true;
    };
  }, [index, v]);
  if (index === undefined) return null;
  return {
    opacity: v,
    transform: [{ translateY: v.interpolate({ inputRange: [0, 1], outputRange: [48, 0] }) }, { scale: v.interpolate({ inputRange: [0, 1], outputRange: [0.96, 1] }) }],
  };
}

// Web-only: track the pointer in CSS variables so a radial glow follows it.
function useSpotlight(enabled: boolean) {
  const ref = useRef<View>(null);
  useEffect(() => {
    if (Platform.OS !== 'web' || !enabled) return;
    const el = ref.current as unknown as HTMLElement | null;
    if (!el || !window.matchMedia?.('(pointer: fine)').matches) return;
    const onMove = (e: PointerEvent) => {
      const r = el.getBoundingClientRect();
      el.style.setProperty('--spot-x', `${e.clientX - r.left}px`);
      el.style.setProperty('--spot-y', `${e.clientY - r.top}px`);
    };
    el.addEventListener('pointermove', onMove);
    return () => el.removeEventListener('pointermove', onMove);
  }, [enabled]);
  return ref;
}

// The site's bento tile: rounded-card, 1px hairline border, soft layered
// shadow that deepens on hover, spotlight glow, optional entrance animation.
export function Card({ tone = 'default', children, style, padding, onPress, accessibilityLabel, decoration, reveal, spotlight = true }: Props) {
  const { width } = useWindowDimensions();
  // p-5 on phones, p-6 from md, like the site's tiles.
  padding ??= width >= 768 ? 24 : 20;
  const [hovered, setHovered] = useState(false);
  const [pressed, setPressed] = useState(false);
  const revealStyle = useReveal(reveal);
  const ref = useSpotlight(spotlight);
  const base: ViewStyle = {
    backgroundColor: toneBg[tone],
    borderRadius: radius.card,
    borderWidth: 1,
    borderColor: tone === 'ink' ? colors.ink : colors.line,
    padding,
    overflow: 'hidden',
    boxShadow: hovered ? shadow.cardHover : shadow.card,
    ...(Platform.OS === 'web' ? ({ transitionProperty: 'box-shadow', transitionDuration: '500ms', isolation: 'isolate' } as object) : null),
  };
  const glow =
    Platform.OS === 'web' && spotlight
      ? createElement('span', {
          'aria-hidden': true,
          style: {
            position: 'absolute',
            inset: 0,
            pointerEvents: 'none',
            opacity: hovered ? 1 : 0,
            transition: 'opacity 500ms',
            background: `radial-gradient(420px circle at var(--spot-x, 50%) var(--spot-y, 50%), ${tone === 'ink' ? 'rgba(255,255,255,0.08)' : 'rgba(91,76,245,0.08)'}, transparent 45%)`,
          },
        })
      : null;
  const content = (
    <>
      {decoration ? (
        <View pointerEvents="none" style={StyleSheet.absoluteFill}>
          {decoration}
        </View>
      ) : null}
      {glow}
      {children}
    </>
  );
  const hover = { onHoverIn: () => setHovered(true), onHoverOut: () => setHovered(false) };
  if (!onPress) {
    // Hover handlers on a View need a Pressable on web; keep it non-interactive.
    return (
      <AnimatedPressable ref={ref} {...hover} accessible={false} style={[base, revealStyle, style]}>
        {content}
      </AnimatedPressable>
    );
  }
  return (
    <AnimatedPressable
      ref={ref}
      onPress={onPress}
      onPressIn={() => setPressed(true)}
      onPressOut={() => setPressed(false)}
      {...hover}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      style={[base, revealStyle, pressed && { opacity: 0.92 }, style]}
    >
      {content}
    </AnimatedPressable>
  );
}
