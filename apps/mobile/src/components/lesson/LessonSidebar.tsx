import { useRouter, type Href } from 'expo-router';
import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, View } from 'react-native';

import { GoalBadge, ProgressBar } from '@/components/ui/Bits';
import { Icon, type IconName } from '@/components/ui/Icon';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { RoadmapNode } from '@/lib/api';
import { useDashboard, useRoadmap } from '@/lib/queries';
import { colors, fonts, radius } from '@/theme/tokens';

function Row({ course, node, current }: { course: string; node: RoadmapNode; current: boolean }) {
  const router = useRouter();
  const t = useT();
  const [hovered, setHovered] = useState(false);
  const soon = node.contentStatus !== 'ready';
  const locked = node.status === 'locked';
  const mastered = node.status === 'mastered';
  return (
    <Pressable
      onPress={() => router.replace(locked || soon ? `/courses/${course}/${node.id}` : `/learn/${course}/${node.id}`)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      accessibilityState={{ selected: current }}
      accessibilityLabel={`${node.title}, ${t(`status.${node.status}`)}`}
      style={[styles.row, (current || hovered) && { backgroundColor: current ? colors.inkSoft : colors.canvas }]}
    >
      <View
        style={[
          styles.dot,
          mastered && { backgroundColor: colors.emerald, borderColor: colors.emerald },
          current && !mastered && { borderColor: colors.accent, backgroundColor: colors.accent },
          soon && { borderStyle: 'dashed' },
        ]}
      >
        {mastered ? <Icon name="check" size={12} color={colors.white} strokeWidth={2.5} /> : current ? <View style={styles.inner} /> : null}
      </View>
      <Text size={14} weight={current ? 'medium' : 'regular'} color={locked || soon ? colors.muted : colors.ink} numberOfLines={1} style={{ flex: 1 }}>
        {node.title}
      </Text>
      {node.recommended && !mastered && !soon ? <GoalBadge compact /> : null}
      {soon ? (
        <Text size={11} color={colors.muted} style={{ fontFamily: fonts.mono }}>
          {t('lesson.soon')}
        </Text>
      ) : locked ? (
        <Icon name="lock" size={14} color={colors.muted} />
      ) : null}
    </Pressable>
  );
}

function Link({ icon, label, href, badge }: { icon: IconName; label: string; href: Href; badge?: number }) {
  const router = useRouter();
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={() => router.push(href)}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="link"
      style={[styles.link, hovered && { backgroundColor: colors.canvas }]}
    >
      <Icon name={icon} size={17} color={colors.ink} />
      <Text size={14} style={{ flex: 1 }}>
        {label}
      </Text>
      {badge ? (
        <View style={styles.badge}>
          <Text size={11} weight="medium" color={colors.accent} tabular>
            {badge}
          </Text>
        </View>
      ) : null}
    </Pressable>
  );
}

// Left column of the lesson page: the course's lessons by level, overall
// progress, and links to the roadmap, flashcard review and saved notes.
export function LessonSidebar({ course, nodeId }: { course: string; nodeId: string }) {
  const router = useRouter();
  const t = useT();
  const { data } = useRoadmap(course);
  const dash = useDashboard();
  const ready = data?.nodes.filter((n) => n.contentStatus === 'ready') ?? [];
  const done = ready.filter((n) => n.status === 'mastered').length;
  const levels = (data?.summary.levels ?? []).filter((l) => l.ready > 0);
  return (
    <View style={styles.side}>
      <Pressable onPress={() => router.replace('/dashboard')} accessibilityRole="link" style={styles.back}>
        <Icon name="arrowLeft" size={16} color={colors.muted} />
        <Text size={14} color={colors.muted}>
          {t('lesson.dashboard')}
        </Text>
      </Pressable>
      <ScrollView contentContainerStyle={{ padding: 16, gap: 18 }}>
        {data ? (
          <View style={{ gap: 8, paddingHorizontal: 4 }}>
            <Text weight="semibold" size={18} tight>
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
        {levels.map((l) => {
          const nodes = data!.nodes.filter((n) => n.level === l.level);
          return (
            <View key={l.level} style={{ gap: 2 }}>
              <View style={styles.levelHead}>
                <Eyebrow>{t('lesson.levelLine', { level: l.level, title: l.title })}</Eyebrow>
                <Eyebrow>{`${l.mastered}/${nodes.length}`}</Eyebrow>
              </View>
              {nodes.map((n) => (
                <Row key={n.id} course={course} node={n} current={n.id === nodeId} />
              ))}
            </View>
          );
        })}
      </ScrollView>
      <View style={styles.links}>
        <Link icon="map" label={t('lesson.seeRoadmap')} href={`/courses/${course}`} />
        <Link icon="layers" label={t('lesson.reviewCards')} href="/review" badge={dash.data?.flashcards.due} />
        <Link icon="book" label={t('lesson.notes')} href="/notes" />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  side: { width: 300, borderRightWidth: 1, borderRightColor: colors.line, backgroundColor: colors.card },
  back: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 20, minHeight: 60 },
  levelHead: { flexDirection: 'row', justifyContent: 'space-between', paddingHorizontal: 4, marginBottom: 6 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingHorizontal: 10, paddingVertical: 9, borderRadius: radius.xl },
  dot: { width: 20, height: 20, borderRadius: 10, borderWidth: 1.5, borderColor: colors.lineStrong, alignItems: 'center', justifyContent: 'center' },
  inner: { width: 6, height: 6, borderRadius: 3, backgroundColor: colors.white },
  links: { borderTopWidth: 1, borderTopColor: colors.line, padding: 12, gap: 2 },
  link: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingHorizontal: 12, paddingVertical: 10, borderRadius: radius.xl },
  badge: { minWidth: 22, height: 22, borderRadius: 11, paddingHorizontal: 6, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
});
