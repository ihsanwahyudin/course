import type React from 'react';

import { colors } from '@/theme/tokens';

// The site's `text-gradient-flow`: a mirrored violet → pink → orange gradient
// at twice the width, drifting slowly along the words (still for people who
// ask their device to reduce motion).
const css = `
@keyframes fp-gradient-flow { to { background-position: 100% 50%; } }
.fp-gradient-text {
  background-image: linear-gradient(100deg, ${colors.accent} 0%, #c04cf5 25%, #f5844c 50%, #c04cf5 75%, ${colors.accent} 100%);
  background-size: 200% 100%;
  background-position: 0% 50%;
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  -webkit-text-fill-color: transparent;
  animation: fp-gradient-flow 6s ease-in-out infinite alternate;
}
@media (prefers-reduced-motion: reduce) { .fp-gradient-text { animation: none; } }
`;

function ensureStyle() {
  if (typeof document === 'undefined' || document.getElementById('fp-gradient-text')) return;
  const el = document.createElement('style');
  el.id = 'fp-gradient-text';
  el.textContent = css;
  document.head.appendChild(el);
}

// Inline run inside a Heading with the moving gradient.
export function GradientText({ children }: { children: React.ReactNode }) {
  ensureStyle();
  return <span className="fp-gradient-text">{children}</span>;
}
