import { useState, type ReactNode } from 'react';
import { Platform, Pressable, StyleSheet, useWindowDimensions, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Lesson, StepKind } from '@/lib/api';
import { colors, fonts, radius } from '@/theme/tokens';

export function useKindLabel() {
  const t = useT();
  return (k: StepKind) => t(`step.${k}`);
}

// "① LANGKAH 1 DARI 8 · KONSEP"
export function StepLabel({ n, total, kind, suffix }: { n: number; total: number; kind: StepKind; suffix?: string }) {
  const t = useT();
  const kindLabel = useKindLabel();
  return (
    <View style={styles.label} accessibilityRole="header">
      <View style={styles.num}>
        <Text size={11} color={colors.white} style={{ fontFamily: fonts.monoMedium }}>
          {n}
        </Text>
      </View>
      <Eyebrow color={colors.ink}>{suffix ?? t('step.label', { n, total, kind: kindLabel(kind) })}</Eyebrow>
    </View>
  );
}

// The tutor's line: sparkle avatar + text.
export function TutorLine({ children }: { children: ReactNode }) {
  return (
    <View style={styles.tutor}>
      <View style={styles.avatar} accessible={false}>
        <Icon name="sparkle" size={14} color={colors.accent} />
      </View>
      <View style={{ flex: 1, paddingTop: 2 }}>{typeof children === 'string' ? <Text size={16} style={{ lineHeight: 26 }}>{children}</Text> : children}</View>
    </View>
  );
}

// Content indented under the tutor's avatar (full width on phones).
export function Indented({ children, style }: { children: ReactNode; style?: object }) {
  const { width } = useWindowDimensions();
  return <View style={[{ paddingLeft: width < 600 ? 0 : 44, gap: 12 }, style]}>{children}</View>;
}

// A finished step, collapsed to one row; tap to look at it again.
export function DoneStep({ n, kind, title, children }: { n: number; kind: StepKind; title: string; children: ReactNode }) {
  const t = useT();
  const kindLabel = useKindLabel();
  const [open, setOpen] = useState(false);
  const [hovered, setHovered] = useState(false);
  return (
    <View style={[styles.done, hovered && !open && { backgroundColor: colors.canvas }]}>
      <Pressable
        onPress={() => setOpen((o) => !o)}
        onHoverIn={() => setHovered(true)}
        onHoverOut={() => setHovered(false)}
        accessibilityRole="button"
        accessibilityState={{ expanded: open }}
        style={styles.doneRow}
      >
        <View style={styles.check}>
          <Icon name="check" size={12} color={colors.white} strokeWidth={2.5} />
        </View>
        <Eyebrow color={colors.emeraldDeep}>{t('step.doneLabel', { n, kind: kindLabel(kind) })}</Eyebrow>
        <Text size={14} color={colors.muted} numberOfLines={1} style={{ flex: 1 }}>
          {title}
        </Text>
        <View style={{ transform: [{ rotate: open ? '90deg' : '0deg' }] }}>
          <Icon name="chevronRight" size={16} color={colors.muted} />
        </View>
      </Pressable>
      {open ? <View style={{ padding: 14, paddingTop: 0, gap: 12 }}>{children}</View> : null}
    </View>
  );
}

// Dashed placeholder for the next, still-hidden step.
export function UpcomingStep({ n }: { n: number }) {
  const t = useT();
  return (
    <View style={styles.upcoming}>
      <Icon name="lock" size={14} color={colors.muted} />
      <Text size={13} color={colors.muted}>
        {t('step.upcoming', { n })}
      </Text>
    </View>
  );
}

// "Dari Alfabet & Bunyi: kuat di ejaan · masih ragu bunyi E vs I"
export function MemoryChip({ chip }: { chip: NonNullable<Lesson['memoryChip']> }) {
  const t = useT();
  const lower = (x: string) => x.charAt(0).toLowerCase() + x.slice(1).replace(/\.$/, '');
  const parts = [chip.strong && t('step.memoryStrong', { x: lower(chip.strong) }), chip.weak && t('step.memoryWeak', { x: lower(chip.weak) })].filter(Boolean).join(' · ');
  return (
    <View style={styles.chip}>
      <Icon name="brain" size={14} color={colors.accent} />
      <Text size={13} color={colors.muted} numberOfLines={2} style={{ flexShrink: 1 }}>
        {t('step.memoryFrom', { lesson: chip.from, text: parts })}
      </Text>
    </View>
  );
}

// Row of actions under a step, right-aligned like the wireframes.
export function ActionRow({ left, children }: { left?: ReactNode; children: ReactNode }) {
  return (
    <View style={styles.actions}>
      {left ? <View style={{ flexShrink: 1 }}>{left}</View> : null}
      <View style={{ flex: 1 }} />
      <View style={{ flexDirection: 'row', gap: 8, flexWrap: 'wrap', justifyContent: 'flex-end' }}>{children}</View>
    </View>
  );
}

// Plain text-link button (e.g. "Menyerah, lihat jawaban").
export function LinkButton({ label, icon, onPress, color = colors.muted }: { label: string; icon?: Parameters<typeof Icon>[0]['name']; onPress: () => void; color?: string }) {
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      style={[styles.link, Platform.OS === 'web' && ({ cursor: 'pointer' } as object)]}
    >
      {icon ? <Icon name={icon} size={16} color={hovered ? colors.ink : color} /> : null}
      <Text size={14} color={hovered ? colors.ink : color} style={hovered ? { textDecorationLine: 'underline' } : undefined}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  label: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  num: { width: 22, height: 22, borderRadius: 11, backgroundColor: colors.ink, alignItems: 'center', justifyContent: 'center' },
  tutor: { flexDirection: 'row', gap: 12, alignItems: 'flex-start' },
  avatar: { width: 32, height: 32, borderRadius: 16, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
  done: { borderWidth: 1, borderColor: colors.line, borderRadius: radius.xxl, backgroundColor: colors.card, overflow: 'hidden' },
  doneRow: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingHorizontal: 12, minHeight: 44 },
  check: { width: 22, height: 22, borderRadius: 11, backgroundColor: colors.emerald, alignItems: 'center', justifyContent: 'center' },
  upcoming: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderColor: colors.lineStrong,
    borderRadius: radius.xxl,
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    alignSelf: 'center',
    maxWidth: '100%',
    backgroundColor: colors.card,
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: radius.pill,
    paddingHorizontal: 14,
    paddingVertical: 7,
  },
  actions: { flexDirection: 'row', alignItems: 'center', gap: 12, flexWrap: 'wrap' },
  link: { flexDirection: 'row', alignItems: 'center', gap: 6, paddingVertical: 8 },
});
