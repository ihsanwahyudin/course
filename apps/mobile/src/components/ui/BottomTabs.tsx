import { usePathname, useRouter, type Href } from 'expo-router';
import { Platform, Pressable, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useDashboard } from '@/lib/queries';
import { useT, type TKey } from '@/i18n';
import { colors, fonts, radius } from '@/theme/tokens';
import { BOTTOM_TAB_HEIGHT } from './chrome';
import { Icon, type IconName } from './Icon';
import { Text } from './Text';

const tabs: { href: Href; label: TKey; icon: IconName; match: (p: string) => boolean }[] = [
  { href: '/dashboard', label: 'nav.home', icon: 'home', match: (p) => p === '/dashboard' },
  { href: '/courses', label: 'nav.courses', icon: 'book', match: (p) => p.startsWith('/courses') },
  { href: '/progress', label: 'nav.progress', icon: 'chart', match: (p) => p.startsWith('/progress') },
  { href: '/account', label: 'nav.account', icon: 'user', match: (p) => p.startsWith('/account') },
];

function Tab({ label, icon, active, onPress }: { label: string; icon: IconName; active: boolean; onPress: () => void }) {
  return (
    <Pressable onPress={onPress} accessibilityRole="tab" accessibilityState={{ selected: active }} accessibilityLabel={label} style={styles.tab}>
      <Icon name={icon} size={22} color={active ? colors.ink : colors.muted} strokeWidth={active ? 2 : 1.75} />
      <Text size={11} color={active ? colors.ink : colors.muted} style={{ fontFamily: active ? fonts.semibold : fonts.regular, marginTop: 3 }}>
        {label}
      </Text>
    </Pressable>
  );
}

// Native-style bottom tab bar with a raised ink action in the middle, after
// the Rs app mockups. The middle button resumes learning.
export function BottomTabs() {
  const router = useRouter();
  const pathname = usePathname();
  const insets = useSafeAreaInsets();
  const { data } = useDashboard();
  const t = useT();

  const resume = () => {
    const open = data?.openSessions[0];
    const next = data?.courses.find((c) => c.progress?.nextNode)?.progress?.nextNode;
    const course = data?.courses.find((c) => c.progress?.nextNode)?.slug;
    if (open) router.push(`/learn/${open.courseSlug}/${open.nodeId}`);
    else if (next && course) router.push(`/learn/${course}/${next.id}`);
    else router.navigate('/courses');
  };

  return (
    <View style={[styles.bar, { paddingBottom: Math.max(insets.bottom, 8) }]} accessibilityRole="tablist">
      <Tab {...tabs[0]} label={t(tabs[0].label)} active={tabs[0].match(pathname)} onPress={() => router.navigate(tabs[0].href)} />
      <Tab {...tabs[1]} label={t(tabs[1].label)} active={tabs[1].match(pathname)} onPress={() => router.navigate(tabs[1].href)} />
      <View style={styles.tab}>
        <Pressable onPress={resume} accessibilityRole="button" accessibilityLabel={t('nav.continueLearning')} style={({ pressed }) => [styles.fab, pressed && { transform: [{ scale: 0.94 }] }]}>
          <Icon name="play" size={20} color={colors.white} strokeWidth={2} />
        </Pressable>
      </View>
      <Tab {...tabs[2]} label={t(tabs[2].label)} active={tabs[2].match(pathname)} onPress={() => router.navigate(tabs[2].href)} />
      <Tab {...tabs[3]} label={t(tabs[3].label)} active={tabs[3].match(pathname)} onPress={() => router.navigate(tabs[3].href)} />
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingTop: 8,
    minHeight: BOTTOM_TAB_HEIGHT,
    backgroundColor: colors.card,
    borderTopWidth: 1,
    borderTopColor: colors.line,
    zIndex: 50,
    ...(Platform.OS === 'web' ? ({ position: 'fixed' } as object) : null),
  },
  tab: { flex: 1, alignItems: 'center', justifyContent: 'center', minHeight: 48 },
  fab: {
    width: 52,
    height: 52,
    borderRadius: radius.pill,
    backgroundColor: colors.ink,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: -22,
    borderWidth: 4,
    borderColor: colors.card,
    boxShadow: '0px 8px 20px -6px rgba(17,17,19,0.45)',
  },
});
