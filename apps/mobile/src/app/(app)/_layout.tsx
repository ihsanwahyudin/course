import { Redirect, Stack, usePathname } from 'expo-router';
import { Platform } from 'react-native';

import { useAuth } from '@/lib/auth';
import { colors } from '@/theme/tokens';

// Everything in (app) needs a signed-in user; others go to login and come
// back afterwards via ?next=.
export default function AppLayout() {
  const status = useAuth((s) => s.status);
  const pathname = usePathname();
  if (status !== 'authenticated') {
    // On a fresh page load the router's pathname may not be resolved yet in
    // this layout, so on web take the requested URL from the address bar.
    const next = Platform.OS === 'web' && typeof window !== 'undefined' ? window.location.pathname + window.location.search : pathname;
    return <Redirect href={{ pathname: '/login', params: { next } }} />;
  }
  return <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.canvas }, animation: 'fade' }} />;
}
