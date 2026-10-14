import { usePathname } from 'expo-router';
import { useWindowDimensions } from 'react-native';

import { useAuth } from '@/lib/auth';
import { layout } from '@/theme/tokens';

export const BOTTOM_TAB_HEIGHT = 64;

// Which navigation chrome is on screen. Phones get native-style bottom tabs
// once logged in (like the Rs apps); desktop and logged-out visitors get the
// floating pill nav from ihsan.my.id. Lessons and onboarding are full-screen
// with neither.
export function useChrome() {
  const pathname = usePathname();
  const { width } = useWindowDimensions();
  const status = useAuth((s) => s.status);
  const wide = width >= layout.wideBreakpoint;
  const focused = pathname.startsWith('/learn') || pathname.startsWith('/onboarding');
  const authed = status === 'authenticated';
  return {
    bottomTabs: !focused && authed && !wide,
    topNav: !focused && (wide || !authed),
  };
}
