import type React from 'react';

import { Span } from '@/components/ui/Text';
import { colors } from '@/theme/tokens';

// Native text can't clip a gradient inline, so the phrase keeps the accent
// colour there; the web version (GradientText.web.tsx) has the site's gradient.
export function GradientText({ children }: { children: React.ReactNode }) {
  return <Span color={colors.accent}>{children}</Span>;
}
