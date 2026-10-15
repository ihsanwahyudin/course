import { useRouter } from 'expo-router';
import { Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Session } from '@/lib/api';
import { colors, radius } from '@/theme/tokens';
import { Indented, TutorLine } from './StepParts';

// A level checkpoint has no steps: it shows what it covers, grouped by its
// objectives, with the learner's progress on each lesson.
export function LevelIntro({ session }: { session: Session }) {
  const t = useT();
  const router = useRouter();
  const level = session.level;
  if (!level) return null;
  return (
    <View style={{ gap: 16 }}>
      <TutorLine>{t('level.intro', { level: session.node.level })}</TutorLine>
      <Indented>
        <View style={styles.card}>
          <Eyebrow color={colors.accent}>{t('level.eyebrow', { level: session.node.level })}</Eyebrow>
          <Heading size={24}>{session.node.title}</Heading>
          {session.node.canDo ? (
            <Text size={15} color={colors.muted} style={{ lineHeight: 23 }}>
              {session.node.canDo}
            </Text>
          ) : null}
          {session.node.objectives.map((o) => (
            <View key={o.id} style={{ gap: 8 }}>
              <Text size={15} weight="medium">
                {o.text}
              </Text>
              <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
                {level.lessons
                  .filter((l) => l.objective === o.id)
                  .map((l) => (
                    <Pressable
                      key={l.id}
                      onPress={() => router.push(`/learn/${session.courseSlug}/${l.id}`)}
                      accessibilityRole="link"
                      accessibilityLabel={`${l.title}, ${t(`status.${l.status}`)}`}
                      style={[styles.lesson, l.status === 'mastered' && styles.done]}
                    >
                      {l.status === 'mastered' ? <Icon name="check" size={13} color={colors.emeraldDeep} strokeWidth={2.5} /> : null}
                      <Text size={13} color={l.status === 'mastered' ? colors.emeraldDeep : colors.ink}>
                        {l.title}
                      </Text>
                    </Pressable>
                  ))}
              </View>
            </View>
          ))}
        </View>
      </Indented>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: 24, padding: 20, gap: 14 },
  lesson: { flexDirection: 'row', alignItems: 'center', gap: 5, borderWidth: 1, borderColor: colors.line, borderRadius: radius.pill, paddingHorizontal: 11, paddingVertical: 6 },
  done: { backgroundColor: colors.mint, borderColor: colors.mint },
});
