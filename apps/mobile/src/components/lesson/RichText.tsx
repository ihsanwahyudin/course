import { Text as RNText } from 'react-native';

import { Text } from '@/components/ui/Text';
import { colors, fonts } from '@/theme/tokens';

type Props = { children: string; size?: number; color?: string; weight?: 'regular' | 'medium'; lineHeight?: number };

// Course text supports **bold**, *italic*, `code`, line breaks and "- "
// bullet lines; everything else is plain text (no HTML or links), so
// content can't inject markup.
export function RichText({ children, size = 16, color = colors.ink, weight = 'regular', lineHeight = 1.65 }: Props) {
  const parts = children.replace(/^- /gm, '•  ').split(/(\*\*[^*]+\*\*|`[^`]+`|\*[^*]+\*)/g).filter(Boolean);
  return (
    <Text size={size} color={color} weight={weight} style={{ lineHeight: Math.round(size * lineHeight) }}>
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
