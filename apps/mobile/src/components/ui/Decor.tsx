import { LinearGradient } from 'expo-linear-gradient';
import { useEffect, useId, useState } from 'react';
import { Animated, Easing, Platform, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';
import Svg, { Circle, Defs, LinearGradient as SvgLinearGradient, Path, Pattern, RadialGradient, Rect, Stop } from 'react-native-svg';

import { colors, radius, shadow } from '@/theme/tokens';

// Dot grid used on the page canvas (24px) and inside tiles (16px).
export function DotPattern({ size = 24, r = 1.2, color = colors.dot, style }: { size?: number; r?: number; color?: string; style?: StyleProp<ViewStyle> }) {
  const id = useId().replace(/:/g, '');
  return (
    <Svg style={[StyleSheet.absoluteFill, style]} width="100%" height="100%">
      <Defs>
        <Pattern id={`dots${id}`} width={size} height={size} patternUnits="userSpaceOnUse">
          <Circle cx={size / 2} cy={size / 2} r={r} fill={color} />
        </Pattern>
      </Defs>
      <Rect width="100%" height="100%" fill={`url(#dots${id})`} />
    </Svg>
  );
}

function useLoop(duration: number, delay = 0, alternate = true) {
  const v = useState(() => new Animated.Value(0))[0];
  useEffect(() => {
    const anim = alternate
      ? Animated.loop(
          Animated.sequence([
            Animated.timing(v, { toValue: 1, duration, delay, easing: Easing.inOut(Easing.ease), useNativeDriver: Platform.OS !== 'web' }),
            Animated.timing(v, { toValue: 0, duration, easing: Easing.inOut(Easing.ease), useNativeDriver: Platform.OS !== 'web' }),
          ]),
        )
      : Animated.loop(Animated.timing(v, { toValue: 1, duration, delay, easing: Easing.bezier(0, 0, 0.2, 1), useNativeDriver: Platform.OS !== 'web' }));
    anim.start();
    return () => anim.stop();
  }, [v, duration, delay, alternate]);
  return v;
}

// Soft blurred pastel blob (the site's blur-3xl mesh circles), drifting slowly.
export function MeshBlob({ color, size, style, delay = 0 }: { color: string; size: number; style?: StyleProp<ViewStyle>; delay?: number }) {
  const id = useId().replace(/:/g, '');
  const t = useLoop(7000, delay);
  const translateX = t.interpolate({ inputRange: [0, 0.5, 1], outputRange: [0, size * 0.06, -size * 0.05] });
  const translateY = t.interpolate({ inputRange: [0, 0.5, 1], outputRange: [0, -size * 0.04, size * 0.05] });
  const scale = t.interpolate({ inputRange: [0, 0.5, 1], outputRange: [1, 1.1, 0.95] });
  return (
    <Animated.View style={[{ position: 'absolute', width: size, height: size, transform: [{ translateX }, { translateY }, { scale }] }, style]}>
      <Svg width={size} height={size}>
        <Defs>
          <RadialGradient id={`blob${id}`} cx="50%" cy="50%" r="50%">
            <Stop offset="0" stopColor={color} stopOpacity={1} />
            <Stop offset="0.55" stopColor={color} stopOpacity={0.7} />
            <Stop offset="1" stopColor={color} stopOpacity={0} />
          </RadialGradient>
        </Defs>
        <Circle cx={size / 2} cy={size / 2} r={size / 2} fill={`url(#blob${id})`} />
      </Svg>
    </Animated.View>
  );
}

// Pulsing accent dot with concentric rings (the site's "based in" marker).
export function PulseDot({ size = 16, rings = true, color = colors.accent }: { size?: number; rings?: boolean; color?: string }) {
  const t = useLoop(2400, 0, false);
  const scale = t.interpolate({ inputRange: [0, 0.75, 1], outputRange: [1, 2, 2] });
  const opacity = t.interpolate({ inputRange: [0, 0.75, 1], outputRange: [0.4, 0, 0] });
  const box = rings ? size * 11 : size * 2.2;
  return (
    <View style={{ width: box, height: box, alignItems: 'center', justifyContent: 'center' }} pointerEvents="none">
      {rings ? (
        <>
          <View style={[styles.ring, { width: size * 11, height: size * 11, borderColor: 'rgba(91,76,245,0.10)' }]} />
          <View style={[styles.ring, { width: size * 6, height: size * 6, borderColor: 'rgba(91,76,245,0.20)', backgroundColor: 'rgba(91,76,245,0.04)' }]} />
        </>
      ) : null}
      <Animated.View style={[styles.ring, { width: size, height: size, backgroundColor: color, opacity, transform: [{ scale }] }]} />
      <View style={{ width: size, height: size, borderRadius: radius.pill, backgroundColor: color, borderWidth: Math.max(2, size * 0.19), borderColor: colors.card, boxShadow: shadow.dot }} />
    </View>
  );
}

// The site's CodeMark: a glyph stroked with the headline gradient inside a
// gradient ring. FluentPath uses Lucide's "route" glyph (ISC licence).
export function GradientRing({ size = 36, glyph = 'route' }: { size?: number; glyph?: 'route' | 'check' }) {
  const id = useId().replace(/:/g, '');
  const icon = Math.round(size * 0.46);
  return (
    <LinearGradient colors={[colors.orange, colors.magenta, colors.accent, colors.accent]} locations={[0, 0.35, 0.75, 1]} start={{ x: 1, y: 1 }} end={{ x: 0, y: 0 }} style={{ width: size, height: size, borderRadius: radius.pill, padding: 2 }}>
      <View style={{ flex: 1, borderRadius: radius.pill, backgroundColor: colors.card, alignItems: 'center', justifyContent: 'center' }}>
        <Svg width={icon} height={icon} viewBox="0 0 24 24" fill="none" stroke={`url(#mark${id})`} strokeWidth={2.1} strokeLinecap="round" strokeLinejoin="round">
          <Defs>
            <SvgLinearGradient id={`mark${id}`} x1="0" y1="0" x2="24" y2="24" gradientUnits="userSpaceOnUse">
              <Stop offset="0" stopColor={colors.accent} />
              <Stop offset="0.55" stopColor={colors.magenta} />
              <Stop offset="1" stopColor={colors.orange} />
            </SvgLinearGradient>
          </Defs>
          {glyph === 'route' ? (
            <>
              <Circle cx="6" cy="19" r="3" />
              <Path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15" />
              <Circle cx="18" cy="5" r="3" />
            </>
          ) : (
            <Path d="M20 6 9 17l-5-5" />
          )}
        </Svg>
      </View>
    </LinearGradient>
  );
}

// Small live indicator: solid dot with a slow ping (the site's "Currently" dot).
export function LiveDot({ color = colors.emerald, size = 8 }: { color?: string; size?: number }) {
  const t = useLoop(2400, 0, false);
  const scale = t.interpolate({ inputRange: [0, 0.75, 1], outputRange: [1, 2, 2] });
  const opacity = t.interpolate({ inputRange: [0, 0.75, 1], outputRange: [0.75, 0, 0] });
  return (
    <View style={{ width: size, height: size }}>
      <Animated.View style={[StyleSheet.absoluteFill, { borderRadius: radius.pill, backgroundColor: color, opacity, transform: [{ scale }] }]} />
      <View style={{ width: size, height: size, borderRadius: radius.pill, backgroundColor: color }} />
    </View>
  );
}

const styles = StyleSheet.create({
  ring: { position: 'absolute', borderRadius: radius.pill, borderWidth: 1 },
});
