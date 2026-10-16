import { Platform, View, type TextStyle } from 'react-native';

import { GradientText } from '@/components/ui/GradientText';
import { Heading } from '@/components/ui/Text';
import { colors, fonts } from '@/theme/tokens';

// Hero headline with an inline gradient phrase, like the site's
// "Crafting software solutions in <gradient>backend…</gradient> systems."
// Web flows inline; native stacks the gradient phrase on its own line(s).
export function Headline({ before, gradient, after, size }: { before: string; gradient: string; after: string; size: number }) {
  const gStyle: TextStyle = {
    fontFamily: fonts.semibold,
    fontSize: size,
    lineHeight: Math.round(size * 1.02),
    letterSpacing: -0.045 * size,
  };
  if (Platform.OS === 'web') {
    return (
      <Heading size={size}>
        {before} <GradientText style={gStyle}>{gradient}</GradientText>
        {after}
      </Heading>
    );
  }
  return (
    <View>
      <Heading size={size}>{before}</Heading>
      <GradientText style={{ ...gStyle, color: colors.accent }}>{gradient}</GradientText>
      <Heading size={size}>{after.trim()}</Heading>
    </View>
  );
}
