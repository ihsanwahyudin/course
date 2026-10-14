import { BlurView } from 'expo-blur';
import { LinearGradient } from 'expo-linear-gradient';
import { usePathname, useRouter, type Href } from 'expo-router';
import { useState } from 'react';
import { Platform, Pressable, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useAuth } from '@/lib/auth';
import { useDashboard } from '@/lib/queries';
import { colors, gradient, radius, shadow } from '@/theme/tokens';
import { GradientRing } from './Decor';
import { useLayout } from './Screen';
import { Text } from './Text';

type Item = { href: Href; label: string; match: (p: string) => boolean };

const signedInItems: Item[] = [
  { href: '/dashboard', label: 'Dashboard', match: (p) => p === '/dashboard' },
  { href: '/courses', label: 'Courses', match: (p) => p.startsWith('/courses') },
  { href: '/progress', label: 'Progress', match: (p) => p.startsWith('/progress') },
];
const publicItems: Item[] = [{ href: '/courses', label: 'Courses', match: (p) => p.startsWith('/courses') }];

function NavLink({ label, active, onPress }: { label: string; active: boolean; onPress: () => void }) {
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      accessibilityState={{ selected: active }}
      style={[styles.link, active && { backgroundColor: colors.inkSoft }]}
    >
      <Text size={14} color={active || hovered ? colors.ink : colors.muted}>
        {label}
      </Text>
    </Pressable>
  );
}

function AccentPill({ label, onPress }: { label: string; onPress: () => void }) {
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      style={[styles.cta, { backgroundColor: hovered ? colors.ink : colors.accent }]}
    >
      <Text size={14} weight="medium" color={colors.white}>
        {label}
      </Text>
    </Pressable>
  );
}

// Initial in a gradient ring, like the site's CodeMark.
export function Avatar({ name, size = 36 }: { name: string; size?: number }) {
  return (
    <LinearGradient colors={[...gradient]} start={{ x: 0, y: 1 }} end={{ x: 1, y: 0 }} style={{ width: size, height: size, borderRadius: radius.pill, padding: 2 }}>
      <View style={{ flex: 1, borderRadius: radius.pill, backgroundColor: colors.card, alignItems: 'center', justifyContent: 'center' }}>
        <Text size={size * 0.4} weight="semibold" color={colors.accent}>
          {(name.trim()[0] ?? '?').toUpperCase()}
        </Text>
      </View>
    </LinearGradient>
  );
}

// The site's floating pill navigation: translucent blurred card centred at
// the top. Shown on desktop, and on phones before login (phones get bottom
// tabs once signed in).
export function FloatingNav() {
  const router = useRouter();
  const pathname = usePathname();
  const insets = useSafeAreaInsets();
  const { wide } = useLayout();
  const { status, user } = useAuth();
  const authed = status === 'authenticated';
  const { data } = useDashboard(authed);
  const items = authed ? signedInItems : publicItems;
  const resume = data?.openSessions[0];
  const nextCourse = data?.courses.find((c) => c.progress?.nextNode);
  const onAuthPage = pathname === '/login' || pathname === '/register';

  return (
    <View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + 16 }]}>
      <View style={[styles.shell, { boxShadow: shadow.card }]}>
        <BlurView intensity={40} tint="light" style={StyleSheet.absoluteFill} />
        <View style={[StyleSheet.absoluteFill, { backgroundColor: Platform.OS === 'android' ? 'rgba(255,255,255,0.95)' : 'rgba(255,255,255,0.75)' }]} />
        <Pressable onPress={() => router.navigate(authed ? '/dashboard' : '/')} accessibilityRole="link" accessibilityLabel="FluentPath home">
          <GradientRing size={36} />
        </Pressable>
        {wide ? (
          <View style={styles.links}>
            {items.map((it) => (
              <NavLink key={it.label} label={it.label} active={it.match(pathname)} onPress={() => router.navigate(it.href)} />
            ))}
          </View>
        ) : (
          <Text weight="medium" tight style={{ marginLeft: 4, marginRight: 'auto' }}>
            FluentPath
          </Text>
        )}
        {authed && user ? (
          <>
            {resume || nextCourse ? (
              <AccentPill
                label="Continue"
                onPress={() =>
                  resume
                    ? router.push(`/learn/${resume.courseSlug}/${resume.nodeId}`)
                    : router.push(`/learn/${nextCourse!.slug}/${nextCourse!.progress!.nextNode!.id}`)
                }
              />
            ) : null}
            <Pressable onPress={() => router.navigate('/account')} accessibilityRole="link" accessibilityLabel="Account" style={{ marginLeft: 4 }}>
              <Avatar name={user.displayName} />
            </Pressable>
          </>
        ) : !onAuthPage ? (
          <>
            <NavLink label="Log in" active={false} onPress={() => router.push('/login')} />
            <AccentPill label="Sign up" onPress={() => router.push('/register')} />
          </>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: 'absolute', left: 0, right: 0, alignItems: 'center', paddingHorizontal: 16, zIndex: 50 },
  shell: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    padding: 6,
    borderRadius: radius.pill,
    borderWidth: 1,
    borderColor: colors.line,
    overflow: 'hidden',
    maxWidth: '100%',
  },
  links: { flexDirection: 'row', alignItems: 'center', marginLeft: 4 },
  link: { paddingHorizontal: 14, paddingVertical: 8, borderRadius: radius.pill },
  cta: { marginLeft: 4, borderRadius: radius.pill, paddingHorizontal: 16, paddingVertical: 8 },
});
