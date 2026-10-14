import { Heading, Span } from '@/components/ui/Text';
import { colors } from '@/theme/tokens';

// Hero headline with one phrase picked out in the accent colour.
export function Headline({ before, highlight, after, size }: { before: string; highlight: string; after: string; size: number }) {
  return (
    <Heading size={size}>
      {before} <Span color={colors.accent}>{highlight}</Span>
      {after}
    </Heading>
  );
}
