import { LinearGradient } from 'expo-linear-gradient';
import { useState, type ReactNode } from 'react';
import { ActivityIndicator, Platform, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';

import type { NodeStatus } from '@/lib/api';
import { useT } from '@/i18n';
import { colors, radius } from '@/theme/tokens';
import { Card } from './Card';
import { Icon, type IconName } from './Icon';
import { PillButton } from './PillButton';
import { Eyebrow, Heading, Text } from './Text';

export const statusMeta: Record<NodeStatus, { bg: string; fg: string; dot: string }> = {
  mastered: { bg: colors.mint, fg: colors.emeraldDeep, dot: colors.emerald },
  in_progress: { bg: colors.lavender, fg: colors.accent, dot: colors.accent },
  available: { bg: colors.accentSoft, fg: colors.accent, dot: colors.accent },
  locked: { bg: colors.inkSoft, fg: colors.muted, dot: colors.muted },
  coming_soon: { bg: 'transparent', fg: colors.muted, dot: colors.lineStrong },
};

export function StatusChip({ status, style }: { status: NodeStatus; style?: StyleProp<ViewStyle> }) {
  const m = statusMeta[status];
  const t = useT();
  const label = t(`status.${status}`);
  return (
    <View
      style={[
        styles.chip,
        { backgroundColor: m.bg },
        status === 'coming_soon' && { borderWidth: 1, borderStyle: 'dashed', borderColor: colors.lineStrong },
        style,
      ]}
      accessibilityLabel={label}
    >
      <View style={[styles.chipDot, { backgroundColor: m.dot }]} />
      <Eyebrow color={m.fg}>{label}</Eyebrow>
    </View>
  );
}

// Marks a roadmap node tagged for the learner's onboarding goal. Compact
// shows just the icon (graph cards); otherwise a small labelled pill.
export function GoalBadge({ compact }: { compact?: boolean }) {
  const t = useT();
  const label = t('onboarding.recommended');
  if (compact) {
    return (
      <View style={styles.goalDot} accessibilityLabel={label}>
        <Icon name="target" size={12} color={colors.accent} strokeWidth={2} />
      </View>
    );
  }
  return (
    <View style={[styles.chip, { backgroundColor: colors.accentSoft, alignSelf: 'flex-start' }]} accessibilityLabel={label}>
      <Icon name="target" size={12} color={colors.accent} strokeWidth={2} />
      <Eyebrow color={colors.accent}>{label}</Eyebrow>
    </View>
  );
}

// Bordered white pill used for tags (the site's tech-stack chips).
export function Chip({ label, onPress, icon, active }: { label: string; onPress?: () => void; icon?: IconName; active?: boolean }) {
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={onPress}
      disabled={!onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole={onPress ? 'button' : 'text'}
      style={[
        styles.tag,
        { backgroundColor: active ? colors.ink : hovered && onPress ? colors.canvas : colors.card, borderColor: active ? colors.ink : colors.line },
      ]}
    >
      {icon ? <Icon name={icon} size={14} color={active ? colors.white : colors.muted} /> : null}
      <Text size={13} weight="medium" color={active ? colors.white : colors.ink}>
        {label}
      </Text>
    </Pressable>
  );
}

// Segmented control styled like the floating nav (active item = ink/6% pill).
export function SegmentedPill<T extends string>({ options, value, onChange }: { options: { value: T; label: string; icon?: IconName }[]; value: T; onChange: (v: T) => void }) {
  return (
    <View style={styles.segment} accessibilityRole="tablist">
      {options.map((o) => {
        const on = o.value === value;
        return (
          <Pressable
            key={o.value}
            onPress={() => onChange(o.value)}
            accessibilityRole="tab"
            accessibilityState={{ selected: on }}
            style={[styles.segmentItem, on && { backgroundColor: colors.ink }]}
          >
            {o.icon ? <Icon name={o.icon} size={14} color={on ? colors.white : colors.muted} /> : null}
            <Text size={13} weight="medium" color={on ? colors.white : colors.muted}>
              {o.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

export function ProgressBar({ value, color = colors.accent, track = colors.inkSoft, height = 6 }: { value: number; color?: string; track?: string; height?: number }) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <View
      style={{ height, borderRadius: radius.pill, backgroundColor: track, overflow: 'hidden' }}
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: 100, now: Math.round(pct) }}
    >
      <View style={{ width: `${pct}%`, height: '100%', borderRadius: radius.pill, backgroundColor: color }} />
    </View>
  );
}

export function LoadingState({ label }: { label?: string }) {
  const t = useT();
  return (
    <View style={styles.center}>
      <ActivityIndicator color={colors.accent} />
      <Eyebrow style={{ marginTop: 12 }}>{label ?? t('common.loading')}</Eyebrow>
    </View>
  );
}

export function ErrorState({ title, message, onRetry }: { title?: string; message?: string; onRetry?: () => void }) {
  const t = useT();
  return (
    <Card tone="peach" style={{ gap: 12 }}>
      <Eyebrow color="rgba(17,17,19,0.6)">{t('common.connection')}</Eyebrow>
      <Heading size={24}>{title ?? t('common.somethingWrong')}</Heading>
      {message ? <Text color="rgba(17,17,19,0.7)">{message}</Text> : null}
      {onRetry ? <PillButton label={t('common.tryAgain')} icon="refresh" onPress={onRetry} /> : null}
    </Card>
  );
}

const styles = StyleSheet.create({
  goalDot: { width: 20, height: 20, borderRadius: radius.pill, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: radius.pill,
    alignSelf: 'flex-start',
  },
  chipDot: { width: 6, height: 6, borderRadius: 3 },
  tag: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderWidth: 1,
    borderRadius: radius.pill,
    paddingHorizontal: 12,
    paddingVertical: 7,
    ...(Platform.OS === 'web' ? ({ transitionProperty: 'background-color', transitionDuration: '150ms' } as object) : null),
  },
  segment: {
    flexDirection: 'row',
    alignSelf: 'flex-start',
    gap: 2,
    padding: 4,
    borderRadius: radius.pill,
    backgroundColor: colors.card,
    borderWidth: 1,
    borderColor: colors.line,
  },
  segmentItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: radius.pill,
  },
  eyebrowPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    alignSelf: 'flex-start',
    borderWidth: 1,
    borderColor: colors.line,
    backgroundColor: colors.card,
    borderRadius: radius.pill,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  center: { flex: 1, minHeight: 240, alignItems: 'center', justifyContent: 'center' },
});

// The site's SectionHeader: eyebrow pill with an accent dot, big heading,
// muted description, optional aside on the right (wide screens).
export function SectionHeader({ label, title, description, aside, wide }: { label: string; title: ReactNode; description?: string; aside?: ReactNode; wide: boolean }) {
  return (
    <View style={{ marginBottom: wide ? 40 : 28, gap: 20, flexDirection: wide ? 'row' : 'column', alignItems: wide ? 'flex-end' : 'stretch', justifyContent: 'space-between' }}>
      <View style={{ maxWidth: 672, flexShrink: 1 }}>
        <View style={styles.eyebrowPill}>
          <View style={{ width: 6, height: 6, borderRadius: 3, backgroundColor: colors.accent }} />
          <Eyebrow>{label}</Eyebrow>
        </View>
        <View style={{ marginTop: 20 }}>{typeof title === 'string' ? <Heading size={wide ? 60 : 36}>{title}</Heading> : title}</View>
        {description ? (
          <Text color={colors.muted} size={wide ? 18 : 15} style={{ marginTop: 16, maxWidth: 512 }}>
            {description}
          </Text>
        ) : null}
      </View>
      {aside}
    </View>
  );
}

// Bulleted list with small accent dots (the site's experience highlights).
export function DotList({ items, color = 'rgba(17,17,19,0.75)' }: { items: ReactNode[]; color?: string }) {
  return (
    <View style={{ gap: 12 }}>
      {items.map((it, i) => (
        <View key={i} style={{ flexDirection: 'row', gap: 12 }}>
          <View style={{ width: 6, height: 6, borderRadius: 3, backgroundColor: colors.accent, marginTop: 9 }} />
          <View style={{ flex: 1 }}>{typeof it === 'string' ? <Text color={color} style={{ lineHeight: 24 }}>{it}</Text> : it}</View>
        </View>
      ))}
    </View>
  );
}

// Pagination dots: the active one is a wide gradient pill (the site's
// carousel indicator). Used for the lesson phase stepper.
export function PageDots({ count, active, done = active }: { count: number; active: number; done?: number }) {
  return (
    <View style={{ flexDirection: 'row', gap: 6, alignItems: 'center' }}>
      {Array.from({ length: count }, (_, i) =>
        i === active ? (
          <LinearGradient key={i} colors={[colors.accent, colors.magenta, colors.orange]} start={{ x: 0, y: 0 }} end={{ x: 1, y: 1 }} style={{ width: 24, height: 6, borderRadius: 3 }} />
        ) : (
          <View key={i} style={{ width: 6, height: 6, borderRadius: 3, backgroundColor: i < done ? colors.accent : 'rgba(17,17,19,0.2)' }} />
        ),
      )}
    </View>
  );
}

// "01 / 05" style counter.
export const pad2 = (n: number) => String(n).padStart(2, '0');
