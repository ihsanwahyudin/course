import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, View } from 'react-native';

import { GoalBadge, ProgressBar } from '@/components/ui/Bits';
import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { RoadmapNode } from '@/lib/api';
import { useRoadmap } from '@/lib/queries';
import { colors, fonts, radius } from '@/theme/tokens';

function Row({ course, node, index, current }: { course: string; node: RoadmapNode; index: number; current: boolean }) {
  const router = useRouter();
  const t = useT();
  const [hovered, setHovered] = useState(false);
  const locked = node.status === 'locked';
  return (
    <Pressable
      onPress={() => router.replace(locked ? `/courses/${course}/${node.id}` : `/learn/${course}/${node.id}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      accessibilityState={{ selected: current }}
      accessibilityLabel={`${node.title}, ${t(`status.${node.status}`)}`}
      style={[styles.row, (current || hovered) && { backgroundColor: current ? colors.inkSoft : colors.canvas }]}
    >
      <View style={[styles.num, node.status === 'mastered' && { backgroundColor: colors.emerald, borderColor: colors.emerald }, current && { borderColor: colors.ink }]}>
        {node.status === 'mastered' ? (
          <Icon name="check" size={12} color={colors.white} strokeWidth={2.5} />
        ) : (
          <Text size={11} color={current ? colors.ink : colors.muted} style={{ fontFamily: fonts.mono }}>
            {index + 1}
          </Text>
        )}
      </View>
      <Text size={14} weight={current ? 'medium' : 'regular'} color={locked ? colors.muted : colors.ink} numberOfLines={1} style={{ flex: 1 }}>
        {node.title}
      </Text>
      {node.recommended && node.status !== 'mastered' ? <GoalBadge compact /> : null}
      {locked ? <Icon name="lock" size={14} color={colors.muted} /> : null}
    </Pressable>
  );
}

// Left column of the lesson page: the course's lessons with status and
// overall progress, like a course outline.
export function LessonSidebar({ course, nodeId }: { course: string; nodeId: string }) {
  const router = useRouter();
  const t = useT();
  const { data } = useRoadmap(course);
  const ready = data?.nodes.filter((n) => n.contentStatus === 'ready') ?? [];
  const done = ready.filter((n) => n.status === 'mastered').length;
  return (
    <View style={styles.side}>
      <Pressable onPress={() => router.replace(`/courses/${course}`)} accessibilityRole="link" style={styles.back}>
        <Icon name="arrowLeft" size={16} color={colors.muted} />
        <Text size={14} color={colors.muted} numberOfLines={1}>
          {data?.course.title ?? t('lesson.back')}
        </Text>
      </Pressable>
      <ScrollView contentContainerStyle={{ padding: 16, gap: 16 }}>
        {data ? (
          <View style={{ gap: 8 }}>
            <Text weight="medium" size={17} tight>
              {data.course.title}
            </Text>
            <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
              <Text size={13} color={colors.muted}>
                {t('lesson.progressLine', { done, total: ready.length })}
              </Text>
              <Text size={13} color={colors.muted} tabular>
                {ready.length ? Math.round((done * 100) / ready.length) : 0}%
              </Text>
            </View>
            <ProgressBar value={ready.length ? (done * 100) / ready.length : 0} color={colors.emerald} />
          </View>
        ) : null}
        <View style={{ gap: 2 }}>
          <Eyebrow style={{ marginBottom: 6 }}>{t('lesson.lessons')}</Eyebrow>
          {ready.map((n, i) => (
            <Row key={n.id} course={course} node={n} index={i} current={n.id === nodeId} />
          ))}
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  side: { width: 280, borderRightWidth: 1, borderRightColor: colors.line, backgroundColor: colors.card },
  back: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 16, minHeight: 56, borderBottomWidth: 1, borderBottomColor: colors.line, backgroundColor: colors.canvas },
  row: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingHorizontal: 10, paddingVertical: 9, borderRadius: radius.xl },
  num: { width: 22, height: 22, borderRadius: 11, borderWidth: 1, borderColor: colors.lineStrong, alignItems: 'center', justifyContent: 'center' },
});
