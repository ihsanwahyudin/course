import { BlurView } from 'expo-blur';
import { usePathname, useRouter, type Href } from 'expo-router';
import { useState } from 'react';
import { Platform, Pressable, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useRoadmap } from '@/lib/queries';
import { colors, radius, shadow } from '@/theme/tokens';
import { GradientRing } from './Decor';
import { Icon } from './Icon';
import { useLayout } from './Screen';
import { Text } from './Text';

const items: { href: Href; label: string; match: (p: string) => boolean }[] = [
  { href: '/', label: 'Home', match: (p) => p === '/' },
  { href: '/roadmap', label: 'Roadmap', match: (p) => p.startsWith('/roadmap') || p.startsWith('/nodes') },
  { href: '/progress', label: 'Progress', match: (p) => p.startsWith('/progress') },
];

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

function Cta({ onPress }: { onPress: () => void }) {
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
        Continue
      </Text>
    </Pressable>
  );
}

// The site's floating navigation: a translucent blurred card centred at the
// top. Wide screens show links inline; narrow screens get a menu panel.
export function FloatingNav() {
  const router = useRouter();
  const pathname = usePathname();
  const insets = useSafeAreaInsets();
  const { wide } = useLayout();
  const { data } = useRoadmap();
  const [open, setOpen] = useState(false);
  const next = data?.nextNodeId;
  const go = (href: Href) => {
    setOpen(false);
    router.navigate(href);
  };

  return (
    <View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + 16 }]}>
      <View style={[styles.shell, wide ? styles.shellWide : styles.shellNarrow, { boxShadow: shadow.card }]}>
        <BlurView intensity={40} tint="light" style={StyleSheet.absoluteFill} />
        <View style={[StyleSheet.absoluteFill, { backgroundColor: Platform.OS === 'android' ? 'rgba(255,255,255,0.95)' : 'rgba(255,255,255,0.75)' }]} />
        <View style={styles.bar}>
          <Pressable onPress={() => go('/')} accessibilityRole="link" accessibilityLabel="FluentPath home">
            <GradientRing size={36} />
          </Pressable>
          {wide ? (
            <View style={styles.links}>
              {items.map((it) => (
                <NavLink key={it.label} label={it.label} active={it.match(pathname)} onPress={() => go(it.href)} />
              ))}
            </View>
          ) : (
            <Text weight="medium" tight style={{ marginLeft: 4 }}>
              FluentPath
            </Text>
          )}
          {next ? (
            <View style={{ marginLeft: wide ? 4 : 'auto' }}>
              <Cta onPress={() => go(`/learn/${next}`)} />
            </View>
          ) : null}
          {!wide ? (
            <Pressable
              onPress={() => setOpen((o) => !o)}
              accessibilityRole="button"
              accessibilityLabel={open ? 'Close menu' : 'Open menu'}
              accessibilityState={{ expanded: open }}
              style={[styles.menuBtn, !next && { marginLeft: 'auto' }]}
            >
              <Icon name={open ? 'close' : 'list'} size={18} />
            </Pressable>
          ) : null}
        </View>
        {!wide && open ? (
          <View style={styles.panel}>
            {items.map((it) => {
              const active = it.match(pathname);
              return (
                <Pressable key={it.label} onPress={() => go(it.href)} accessibilityRole="link" style={[styles.panelItem, active && { backgroundColor: colors.inkSoft }]}>
                  <Text size={18} weight="medium" tight>
                    {it.label}
                  </Text>
                  {active ? <View style={styles.activeDot} /> : null}
                </Pressable>
              );
            })}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: 'absolute', left: 0, right: 0, alignItems: 'center', paddingHorizontal: 16, zIndex: 50 },
  shell: { padding: 6, borderWidth: 1, borderColor: colors.line, overflow: 'hidden' },
  shellWide: { borderRadius: radius.pill },
  shellNarrow: { borderRadius: 24, width: '100%', maxWidth: 576 },
  bar: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  links: { flexDirection: 'row', alignItems: 'center' },
  link: { paddingHorizontal: 16, paddingVertical: 8, borderRadius: radius.pill },
  cta: { borderRadius: radius.pill, paddingHorizontal: 16, paddingVertical: 8 },
  menuBtn: { width: 36, height: 36, borderRadius: radius.pill, backgroundColor: colors.inkSoft, alignItems: 'center', justifyContent: 'center' },
  panel: { gap: 4, paddingHorizontal: 4, paddingBottom: 4, paddingTop: 12 },
  panelItem: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', borderRadius: radius.xxl, paddingHorizontal: 16, paddingVertical: 12 },
  activeDot: { width: 6, height: 6, borderRadius: 3, backgroundColor: colors.accent },
});
