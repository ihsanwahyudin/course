import MaskedView from '@react-native-masked-view/masked-view';
import { LinearGradient } from 'expo-linear-gradient';
import { createElement } from 'react';
import { Platform, Text as RNText, type TextStyle } from 'react-native';

import { gradientFlow } from '@/theme/tokens';

type Props = { children: string; style?: TextStyle; animated?: boolean };

// The site's `.text-gradient-flow`: violet → magenta → orange headline text.
// Web uses background-clip:text with the 6s flow animation from global.css;
// native masks a LinearGradient with the text.
export function GradientText({ children, style, animated = true }: Props) {
  if (Platform.OS === 'web') {
    return createElement(
      'span',
      {
        className: animated ? 'fp-gradient-flow' : 'fp-gradient',
        // DOM treats a unitless line-height as a multiplier, so pin px values.
        style: {
          ...(style as object),
          lineHeight: typeof style?.lineHeight === 'number' ? `${style.lineHeight}px` : style?.lineHeight,
          display: 'inline',
        },
      },
      children,
    );
  }
  return (
    <MaskedView maskElement={<RNText style={style}>{children}</RNText>}>
      <LinearGradient colors={[gradientFlow[0], gradientFlow[1], gradientFlow[2]]} start={{ x: 0, y: 0 }} end={{ x: 1, y: 0.4 }}>
        <RNText style={[style, { opacity: 0 }]}>{children}</RNText>
      </LinearGradient>
    </MaskedView>
  );
}
