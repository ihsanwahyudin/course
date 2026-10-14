import { Geist_400Regular, Geist_500Medium, Geist_600SemiBold, useFonts } from '@expo-google-fonts/geist';
import { GeistMono_400Regular, GeistMono_500Medium } from '@expo-google-fonts/geist-mono';
import { QueryClientProvider } from '@tanstack/react-query';
import { Stack, usePathname } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';
import { Platform, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { FloatingNav } from '@/components/ui/FloatingNav';
import { queryClient } from '@/lib/queries';
import { colors } from '@/theme/tokens';

if (Platform.OS === 'web') {
  // Web-only CSS (keyframes, selection, focus ring); native has no stylesheet.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  require('../global.css');
}

SplashScreen.preventAutoHideAsync().catch(() => {});

export default function RootLayout() {
  const [loaded, error] = useFonts({ Geist_400Regular, Geist_500Medium, Geist_600SemiBold, GeistMono_400Regular, GeistMono_500Medium });
  const pathname = usePathname();

  useEffect(() => {
    if (loaded || error) SplashScreen.hideAsync().catch(() => {});
  }, [loaded, error]);

  if (!loaded && !error) return null;

  // The lesson screen is a focused, full-height chat with its own header.
  const showNav = !pathname.startsWith('/learn');

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <StatusBar style="dark" />
        <View style={{ flex: 1, backgroundColor: colors.canvas }}>
          <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.canvas }, animation: 'fade' }} />
          {showNav ? <FloatingNav /> : null}
        </View>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
