import { StyleSheet, View } from 'react-native';

import { Icon, type IconName } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { RoadmapNode, Session } from '@/lib/api';
import { colors, radius } from '@/theme/tokens';
import { AttemptHistory, AttemptReview } from './CheckpointReview';
import { Indented, TutorLine } from './StepParts';

function Chip({ icon, label }: { icon: IconName; label: string }) {
  return (
    <View style={styles.chip}>
      <Icon name={icon} size={14} color={colors.accent} />
      <Text size={13} color={colors.muted} style={{ flexShrink: 1 }}>
        {label}
      </Text>
    </View>
  );
}

// After a passed checkpoint: score per objective with every answer's
// correction, the lesson summary, the checkpoint history and what comes next.
export function LessonResult({
  session,
  unlocked,
  onNext,
  onRoadmap,
  onPractice,
}: {
  session: Session;
  unlocked: RoadmapNode[];
  onNext: (id: string) => void;
  onRoadmap: () => void;
  onPractice?: () => void;
}) {
  const t = useT();
  // The attempt that mastered the lesson (later ones may be practice).
  const a = session.masteredBy ?? (session.lastAttempt && !session.lastAttempt.practice ? session.lastAttempt : null);
  const sum = session.lesson?.summary ?? null;
  const following = session.next && session.next.status !== 'locked' && session.next.status !== 'coming_soon' ? session.next : null;
  const next = sum?.next ?? unlocked[0] ?? following;
  return (
    <View style={{ gap: 16 }}>
      <TutorLine>{t('result.done')}</TutorLine>
      <Indented>
        {a ? (
          <View style={styles.score}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
              <Icon name="check" size={16} color={colors.white} strokeWidth={2.25} />
              <Eyebrow color={colors.white}>{t('result.mastered')}</Eyebrow>
            </View>
            <View style={{ flexDirection: 'row', alignItems: 'flex-end', gap: 4 }} accessibilityLabel={`${a.score}/100`}>
              <Heading size={64} color={colors.white} tabular>
                {a.score}
              </Heading>
              <Text size={22} color="rgba(255,255,255,0.55)" style={{ marginBottom: 6 }}>
                /100
              </Text>
            </View>
            {session.node.objectives.map((o) => {
              const s = a.objectiveScores[o.id] ?? 0;
              return (
                <View key={o.id} style={{ gap: 6 }}>
                  <View style={{ flexDirection: 'row', gap: 12 }}>
                    <Text size={14} color="rgba(255,255,255,0.85)" style={{ flex: 1 }}>
                      {o.text}
                    </Text>
                    <Text size={13} color={colors.white} tabular>
                      {s}
                    </Text>
                  </View>
                  <View style={styles.track}>
                    <View style={{ width: `${s}%`, height: '100%', borderRadius: radius.pill, backgroundColor: s >= a.threshold ? colors.emerald : colors.white }} />
                  </View>
                </View>
              );
            })}
            {unlocked.length ? (
              <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
                {unlocked.map((u) => (
                  <View key={u.id} style={styles.unlocked}>
                    <Text size={14} weight="medium" color={colors.white}>
                      {t('result.unlocked', { title: u.title })}
                    </Text>
                  </View>
                ))}
              </View>
            ) : null}
          </View>
        ) : null}
        {a?.review ? (
          <View style={styles.card}>
            <Eyebrow color={colors.accent}>{t('correction.title')}</Eyebrow>
            <AttemptReview attempt={a} />
          </View>
        ) : null}

        {sum ? (
          <View style={styles.card}>
            <Eyebrow color={colors.accent}>{t('result.summary')}</Eyebrow>
            <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 20 }}>
              <View style={{ flex: 1, minWidth: 220, gap: 8 }}>
                <Text weight="medium">{t('result.masteredCol')}</Text>
                {sum.mastered.map((m, i) => (
                  <View key={i} style={styles.line}>
                    <Icon name="check" size={14} color={colors.emeraldDeep} strokeWidth={2.25} />
                    <Text size={14} style={{ flex: 1 }}>
                      {m}
                    </Text>
                  </View>
                ))}
              </View>
              <View style={{ flex: 1, minWidth: 220, gap: 8 }}>
                <Text weight="medium">{t('result.reviewCol')}</Text>
                {sum.review.length ? (
                  sum.review.map((m, i) => (
                    <View key={i} style={styles.line}>
                      <View style={styles.dot} />
                      <Text size={14} style={{ flex: 1 }}>
                        {m}
                      </Text>
                    </View>
                  ))
                ) : (
                  <Text size={14} color={colors.muted}>
                    {t('result.nothingToReview')}
                  </Text>
                )}
              </View>
            </View>
            {sum.remembered.length || sum.flashcardsAdded ? (
              <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
                {sum.remembered.map((m, i) => (
                  <Chip key={i} icon="brain" label={t('result.remembered', { text: m })} />
                ))}
                {sum.flashcardsAdded ? (
                  <Chip icon="layers" label={sum.flashcardsAdded === 1 ? t('result.mistakeCard') : t('result.mistakeCards', { n: sum.flashcardsAdded })} />
                ) : null}
              </View>
            ) : null}
          </View>
        ) : null}

        <View style={styles.next}>
          {next ? (
            <>
              <Eyebrow color={colors.accent}>{t('result.nextEyebrow')}</Eyebrow>
              <Heading size={26}>{next.title}</Heading>
              {sum?.nextOpening ? <Text size={15} style={{ lineHeight: 24 }}>{`“${sum.nextOpening}”`}</Text> : null}
            </>
          ) : (
            <Text size={15}>{t('result.finished')}</Text>
          )}
          <View style={{ flexDirection: 'row', gap: 10, flexWrap: 'wrap' }}>
            {next && next.status !== 'locked' && next.contentStatus === 'ready' ? (
              <PillButton label={t('result.startNext', { title: next.title })} onPress={() => onNext(next.id)} />
            ) : null}
            <PillButton label={t('result.backRoadmap')} variant="secondary" onPress={onRoadmap} />
            {onPractice ? <PillButton label={t('result.practiceAgain')} icon="refresh" variant="secondary" onPress={onPractice} /> : null}
          </View>
        </View>
        {session.attempts.length > 1 ? <AttemptHistory attempts={session.attempts} current={session.lastAttempt?.id} /> : null}
      </Indented>
    </View>
  );
}

const styles = StyleSheet.create({
  score: { backgroundColor: colors.ink, borderRadius: 24, padding: 22, gap: 14 },
  track: { height: 4, borderRadius: radius.pill, backgroundColor: 'rgba(255,255,255,0.15)', overflow: 'hidden' },
  unlocked: { backgroundColor: 'rgba(255,255,255,0.1)', borderRadius: radius.xl, paddingHorizontal: 14, paddingVertical: 9 },
  card: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: 24, padding: 20, gap: 14 },
  line: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  dot: { width: 7, height: 7, borderRadius: 4, backgroundColor: colors.danger, marginHorizontal: 3 },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 6, backgroundColor: colors.canvas, borderRadius: radius.pill, paddingHorizontal: 12, paddingVertical: 6, maxWidth: '100%' },
  next: { backgroundColor: colors.accentSoft, borderRadius: 24, padding: 22, gap: 12 },
});
