import { Geist_400Regular, Geist_500Medium, Geist_600SemiBold, useFonts } from '@expo-google-fonts/geist';
import { GeistMono_400Regular, GeistMono_500Medium } from '@expo-google-fonts/geist-mono';
import { QueryClientProvider } from '@tanstack/react-query';
import { Stack } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import { StatusBar } from 'expo-status-bar';
import { useEffect, useState } from 'react';
import { Platform, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { BottomTabs } from '@/components/ui/BottomTabs';
import { useChrome } from '@/components/ui/chrome';
import { FloatingNav } from '@/components/ui/FloatingNav';
import { useLocale } from '@/i18n';
import { useAuth } from '@/lib/auth';
import { queryClient } from '@/lib/queries';
import { colors } from '@/theme/tokens';

if (Platform.OS === 'web') {
  // Web-only CSS (keyframes, selection, focus ring); native has no stylesheet.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  require('../global.css');
}

SplashScreen.preventAutoHideAsync().catch(() => {});

function Chrome() {
  const { topNav, bottomTabs } = useChrome();
  return (
    <>
      {topNav ? <FloatingNav /> : null}
      {bottomTabs ? <BottomTabs /> : null}
    </>
  );
}

export default function RootLayout() {
  const [fontsLoaded, fontError] = useFonts({ Geist_400Regular, Geist_500Medium, Geist_600SemiBold, GeistMono_400Regular, GeistMono_500Medium });
  const authStatus = useAuth((s) => s.status);
  const init = useAuth((s) => s.init);
  const restoreLocale = useLocale((s) => s.restore);
  const [localeReady, setLocaleReady] = useState(false);

  // Device language choice first, then auth (a signed-in account's saved
  // language overrides it).
  useEffect(() => {
    void restoreLocale()
      .catch(() => {})
      .finally(() => {
        setLocaleReady(true);
        void init();
      });
  }, [init, restoreLocale]);

  const ready = (fontsLoaded || !!fontError) && localeReady && authStatus !== 'loading';
  useEffect(() => {
    if (ready) SplashScreen.hideAsync().catch(() => {});
  }, [ready]);

  if (!ready) return null;

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <StatusBar style="dark" />
        <View style={{ flex: 1, backgroundColor: colors.canvas }}>
          <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.canvas }, animation: 'fade' }} />
          <Chrome />
        </View>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
