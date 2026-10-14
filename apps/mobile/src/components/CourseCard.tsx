import { useRouter } from 'expo-router';
import { StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';

import { ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { MeshBlob } from '@/components/ui/Decor';
import { Icon } from '@/components/ui/Icon';
import { ArrowBadge } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { Course } from '@/lib/api';
import { colors, radius, toneBg, type Tone } from '@/theme/tokens';

const blobFor: Record<string, string> = {
  lavender: colors.lavender,
  peach: colors.peach,
  mint: colors.mint,
  sky: colors.sky,
};

// A course tile for the catalog and dashboard: a soft pastel mesh in the
// course's tone, title, tagline, size, and progress once enrolled.
export function CourseCard({ course, reveal, style }: { course: Course; reveal?: number; style?: StyleProp<ViewStyle> }) {
  const router = useRouter();
  const p = course.progress;
  const levels = course.levels.length ? `${course.levels[0]}–${course.levels[course.levels.length - 1]}` : '';
  return (
    <Card
      reveal={reveal}
      onPress={() => router.push(`/courses/${course.slug}`)}
      accessibilityLabel={`${course.title} course`}
      style={[{ gap: 20, minHeight: 240, justifyContent: 'space-between' }, style]}
      decoration={<MeshBlob color={blobFor[course.tone] ?? colors.lavender} size={280} style={{ right: -110, top: -120 }} />}
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
              Level {p.currentLevel} · {p.percent}% to {p.targetLevel}
            </Text>
            <Text size={12} color={colors.muted} tabular>
              {p.mastered}/{p.targetNodes}
            </Text>
          </View>
          <ProgressBar value={p.percent} />
          {p.nextNode ? (
            <Text size={13} color={colors.muted} numberOfLines={1}>
              Next: {p.nextNode.title}
            </Text>
          ) : null}
        </View>
      ) : (
        <View style={[styles.row, { flexWrap: 'wrap', gap: 6 }]}>
          <Meta icon="book" label={`${course.readyNodes} lessons ready`} />
          <Meta icon="route" label={`${course.totalNodes} skills`} />
          <Meta icon="target" label={`Goal ${course.targetLevel}`} />
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
  return (
    <View style={[styles.soon, { backgroundColor: tone === 'default' ? 'rgba(255,255,255,0.5)' : toneBg[tone] }, style]}>
      <Eyebrow>Coming soon</Eyebrow>
      <Heading size={24} color={colors.muted}>
        More courses
      </Heading>
      <Text size={13} color={colors.muted}>
        New roadmaps will appear here. English is the first course on FluentPath.
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
