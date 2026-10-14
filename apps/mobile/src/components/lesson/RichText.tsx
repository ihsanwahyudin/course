import { Text as RNText } from 'react-native';

import { Text } from '@/components/ui/Text';
import { colors, fonts } from '@/theme/tokens';

type Props = { children: string; size?: number; color?: string };

// Course text supports **bold** and `code`; everything else is plain text
// (no HTML or links), so content can't inject markup.
export function RichText({ children, size = 16, color = colors.ink }: Props) {
  const parts = children.split(/(\*\*[^*]+\*\*|`[^`]+`|\*[^*]+\*)/g).filter(Boolean);
  return (
    <Text size={size} color={color} style={{ lineHeight: Math.round(size * 1.65) }}>
      {parts.map((p, i) => {
        if (p.startsWith('**') && p.endsWith('**')) {
          return (
            <RNText key={i} style={{ fontFamily: fonts.semibold }}>
              {p.slice(2, -2)}
            </RNText>
          );
        }
        if (p.startsWith('`') && p.endsWith('`')) {
          return (
            <RNText key={i} style={{ fontFamily: fonts.mono, fontSize: size * 0.9, backgroundColor: colors.inkSoft }}>
              {` ${p.slice(1, -1)} `}
            </RNText>
          );
        }
        if (p.length > 2 && p.startsWith('*') && p.endsWith('*')) {
          return (
            <RNText key={i} style={{ fontStyle: 'italic' }}>
              {p.slice(1, -1)}
            </RNText>
          );
        }
        return p;
      })}
    </Text>
  );
}
