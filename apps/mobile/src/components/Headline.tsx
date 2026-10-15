import { GradientText } from '@/components/ui/GradientText';
import { Heading } from '@/components/ui/Text';

// Hero headline with one phrase picked out in the site's gradient.
export function Headline({ before, highlight, after, size }: { before: string; highlight: string; after: string; size: number }) {
  return (
    <Heading size={size}>
      {before} <GradientText>{highlight}</GradientText>
      {after}
    </Heading>
  );
}
