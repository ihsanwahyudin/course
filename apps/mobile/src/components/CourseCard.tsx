import { useRouter } from 'expo-router';
import { StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';

import { ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { ArrowBadge } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { Course } from '@/lib/api';
import { useT } from '@/i18n';
import { colors, radius, toneBg, type Tone } from '@/theme/tokens';

// A course tile for the catalog and dashboard: title, tagline, size, and
// progress once enrolled.
export function CourseCard({ course, reveal, style }: { course: Course; reveal?: number; style?: StyleProp<ViewStyle> }) {
  const router = useRouter();
  const t = useT();
  const p = course.progress;
  const levels = course.levels.length ? `${course.levels[0]}–${course.levels[course.levels.length - 1]}` : '';
  return (
    <Card
      reveal={reveal}
      onPress={() => router.push(`/courses/${course.slug}`)}
      accessibilityLabel={t('catalog.courseLabel', { title: course.title })}
      style={[{ gap: 20, minHeight: 240, justifyContent: 'space-between' }, style]}
    >
      <View style={[styles.row, { justifyContent: 'space-between' }]}>
        <Eyebrow>
          {course.category} · {levels}
        </Eyebrow>
        <ArrowBadge size={34} />
      </View>
      <View style={{ gap: 6 }}>
        <Heading size={36}>{course.title}</Heading>
        <Text color={colors.muted}>{course.tagline}</Text>
      </View>
      {p ? (
        <View style={{ gap: 8 }}>
          <View style={[styles.row, { justifyContent: 'space-between' }]}>
            <Text size={13} weight="medium">
              {t('catalog.levelProgress', { level: p.currentLevel, pct: p.percent, target: p.targetLevel })}
            </Text>
            <Text size={12} color={colors.muted} tabular>
              {p.mastered}/{p.targetNodes}
            </Text>
          </View>
          <ProgressBar value={p.percent} />
          {p.goal ? (
            <View style={styles.row}>
              <Icon name="target" size={13} color={colors.accent} strokeWidth={2} />
              <Text size={13} color={colors.accent} numberOfLines={1}>
                {t('onboarding.focusLine', { goal: p.goal.label })}
              </Text>
            </View>
          ) : null}
          {p.nextNode ? (
            <Text size={13} color={colors.muted} numberOfLines={1}>
              {t('catalog.next', { title: p.nextNode.title })}
            </Text>
          ) : null}
        </View>
      ) : (
        <View style={[styles.row, { flexWrap: 'wrap', gap: 6 }]}>
          <Meta icon="book" label={t('common.lessonsReady', { n: course.readyNodes })} />
          <Meta icon="route" label={t('common.skills', { n: course.totalNodes })} />
          <Meta icon="target" label={t('catalog.goal', { level: course.targetLevel })} />
        </View>
      )}
    </Card>
  );
}

function Meta({ icon, label }: { icon: 'book' | 'route' | 'target'; label: string }) {
  return (
    <View style={styles.meta}>
      <Icon name={icon} size={13} color={colors.muted} />
      <Text size={12} color={colors.ink}>
        {label}
      </Text>
    </View>
  );
}

// Placeholder tile so the catalog reads as a platform with more to come.
export function ComingSoonCard({ style, tone = 'default' }: { style?: StyleProp<ViewStyle>; tone?: Tone }) {
  const t = useT();
  return (
    <View style={[styles.soon, { backgroundColor: tone === 'default' ? 'rgba(255,255,255,0.5)' : toneBg[tone] }, style]}>
      <Eyebrow>{t('common.comingSoon')}</Eyebrow>
      <Heading size={24} color={colors.muted}>
        {t('catalog.moreTitle')}
      </Heading>
      <Text size={13} color={colors.muted}>
        {t('catalog.moreText')}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  meta: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderWidth: 1,
    borderColor: colors.line,
    backgroundColor: 'rgba(243,242,238,0.6)',
    borderRadius: radius.pill,
    paddingHorizontal: 10,
    paddingVertical: 5,
  },
  soon: {
    borderRadius: radius.card,
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: colors.lineStrong,
    padding: 24,
    gap: 8,
    minHeight: 200,
    justifyContent: 'flex-end',
  },
});
