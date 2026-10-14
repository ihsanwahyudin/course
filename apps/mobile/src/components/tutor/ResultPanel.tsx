import { BlurView } from 'expo-blur';
import { LinearGradient } from 'expo-linear-gradient';
import { useState } from 'react';
import { Modal, Pressable, ScrollView, StyleSheet, View } from 'react-native';

import { ProgressBar } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { MeshBlob } from '@/components/ui/Decor';
import { GradientText } from '@/components/ui/GradientText';
import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { AssessmentResult } from '@/lib/api';
import { colors, fonts, radius } from '@/theme/tokens';

function WhitePill({ label, onPress }: { label: string; onPress: () => void }) {
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      style={[styles.whitePill, { backgroundColor: hovered ? colors.accent : colors.white }]}
    >
      <Text weight="medium" color={hovered ? colors.white : colors.ink}>
        {label}
      </Text>
      <View style={{ transform: [{ rotate: hovered ? '45deg' : '0deg' }] }}>
        <Icon name="arrowUpRight" size={16} color={hovered ? colors.white : colors.ink} />
      </View>
    </Pressable>
  );
}

// Assessment result dialog. A pass uses the site's ink "contact" card with
// violet/magenta/orange mesh; a fail uses a calm peach card with a retry path.
export function ResultPanel({
  result,
  onClose,
  onNext,
  onRoadmap,
}: {
  result: AssessmentResult;
  onClose: () => void;
  onNext: (nodeId: string) => void;
  onRoadmap: () => void;
}) {
  const { attempt, passed, unlocked, session } = result;
  const objectives = session.node.objectives;
  const ink = passed;
  const fg = ink ? colors.white : colors.ink;
  const soft = ink ? 'rgba(255,255,255,0.7)' : 'rgba(17,17,19,0.7)';

  return (
    <Modal transparent animationType="fade" onRequestClose={onClose} visible>
      <View style={StyleSheet.absoluteFill}>
        <BlurView intensity={20} tint="dark" style={StyleSheet.absoluteFill} />
        <Pressable style={[StyleSheet.absoluteFill, { backgroundColor: 'rgba(17,17,19,0.35)' }]} onPress={onClose} accessibilityLabel="Close result" />
        <ScrollView contentContainerStyle={styles.scroll} pointerEvents="box-none">
          <Card
            tone={ink ? 'ink' : 'peach'}
            padding={28}
            reveal={0}
            spotlight={false}
            style={{ width: '100%', maxWidth: 560, gap: 24 }}
            decoration={
              ink ? (
                <>
                  <MeshBlob color="rgba(91,76,245,0.6)" size={360} style={{ left: -90, top: -140 }} />
                  <MeshBlob color="rgba(192,76,245,0.4)" size={380} style={{ right: -120, bottom: -180 }} delay={2000} />
                  <MeshBlob color="rgba(245,132,76,0.3)" size={240} style={{ left: 120, bottom: -100 }} delay={4000} />
                </>
              ) : (
                <MeshBlob color="rgba(255,255,255,0.7)" size={300} style={{ right: -100, top: -120 }} />
              )
            }
          >
            <Pressable onPress={onClose} accessibilityRole="button" accessibilityLabel="Close" style={[styles.close, { backgroundColor: ink ? 'rgba(255,255,255,0.1)' : 'rgba(255,255,255,0.7)' }]}>
              <Icon name="close" size={16} color={fg} />
            </Pressable>
            <View style={[styles.eyebrowPill, { borderColor: ink ? 'rgba(255,255,255,0.15)' : colors.line, backgroundColor: ink ? 'rgba(255,255,255,0.05)' : 'rgba(255,255,255,0.6)' }]}>
              <View style={{ width: 6, height: 6, borderRadius: 3, backgroundColor: ink ? colors.white : colors.orange }} />
              <Eyebrow color={soft}>{passed ? (result.alreadyCompleted ? 'Already mastered' : 'Mastered') : 'Review needed'}</Eyebrow>
            </View>
            <View>
              <View style={{ flexDirection: 'row', alignItems: 'flex-end' }}>
                <Heading size={88} color={fg} tabular>
                  {attempt.score}
                </Heading>
                {ink ? (
                  <GradientText animated={false} style={{ fontFamily: fonts.semibold, fontSize: 36, lineHeight: 48, letterSpacing: -1.6 }}>
                    /100
                  </GradientText>
                ) : (
                  <Heading size={36} color="rgba(17,17,19,0.45)" style={{ marginBottom: 6 }}>
                    /100
                  </Heading>
                )}
              </View>
              <Heading size={28} color={fg} style={{ marginTop: 12 }}>
                {passed ? `You mastered ${session.node.title}.` : `Almost there — you need ${attempt.threshold}.`}
              </Heading>
              <Text color={soft} style={{ marginTop: 10 }}>
                {attempt.feedback}
              </Text>
            </View>

            <View style={{ gap: 12 }}>
              {objectives.map((o) => {
                const v = attempt.objectiveScores[o.id] ?? 0;
                return (
                  <View key={o.id} style={{ gap: 6 }}>
                    <View style={{ flexDirection: 'row', justifyContent: 'space-between', gap: 12 }}>
                      <Text size={13} color={soft} style={{ flex: 1 }}>
                        {o.text}
                      </Text>
                      <Text size={12} color={fg} style={{ fontFamily: fonts.mono }}>
                        {v}
                      </Text>
                    </View>
                    {ink ? (
                      <View style={{ height: 5, borderRadius: 3, backgroundColor: 'rgba(255,255,255,0.1)', overflow: 'hidden' }}>
                        <LinearGradient colors={[colors.accent, colors.magenta, colors.orange]} start={{ x: 0, y: 0 }} end={{ x: 1, y: 0 }} style={{ width: `${v}%`, height: '100%' }} />
                      </View>
                    ) : (
                      <ProgressBar value={v} color={v >= attempt.threshold ? colors.emerald : colors.ink} track="rgba(17,17,19,0.1)" height={5} />
                    )}
                  </View>
                );
              })}
            </View>

            {attempt.evidence.length > 0 ? (
              <View style={{ gap: 6 }}>
                {attempt.evidence.slice(0, 4).map((e) => (
                  <View key={e} style={{ flexDirection: 'row', gap: 8 }}>
                    <Icon name={passed ? 'check' : 'sparkle'} size={14} color={passed ? '#00d294' : colors.ink} />
                    <Text size={13} color={soft} style={{ flex: 1 }}>
                      {e}
                    </Text>
                  </View>
                ))}
              </View>
            ) : null}

            {passed && unlocked.length > 0 ? (
              <View style={{ gap: 8 }}>
                <Eyebrow color="rgba(255,255,255,0.6)">Unlocked</Eyebrow>
                <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
                  {unlocked.map((n) => (
                    <View key={n.id} style={styles.inkChip}>
                      <Text size={13} color={colors.white}>
                        {n.level} · {n.title}
                      </Text>
                    </View>
                  ))}
                </View>
              </View>
            ) : null}

            <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
              {passed ? (
                <>
                  {unlocked[0] ? <WhitePill label={`Next: ${unlocked[0].title}`} onPress={() => onNext(unlocked[0].id)} /> : <WhitePill label="Back to roadmap" onPress={onRoadmap} />}
                  {unlocked[0] ? (
                    <Pressable onPress={onRoadmap} accessibilityRole="button" style={styles.ghost}>
                      <Text size={14} color={colors.white} style={{ fontFamily: fonts.mono }}>
                        View roadmap
                      </Text>
                    </Pressable>
                  ) : null}
                </>
              ) : (
                <>
                  <PillButton label="Review with tutor" icon="arrowUpRight" onPress={onClose} />
                  <PillButton label="Back to roadmap" variant="secondary" onPress={onRoadmap} />
                </>
              )}
            </View>
          </Card>
        </ScrollView>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  scroll: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 12, paddingVertical: 40 },
  close: { position: 'absolute', right: 18, top: 18, width: 38, height: 38, borderRadius: radius.pill, alignItems: 'center', justifyContent: 'center', zIndex: 2 },
  eyebrowPill: { flexDirection: 'row', alignItems: 'center', gap: 8, alignSelf: 'flex-start', borderWidth: 1, borderRadius: radius.pill, paddingHorizontal: 12, paddingVertical: 6 },
  whitePill: { flexDirection: 'row', alignItems: 'center', gap: 8, borderRadius: radius.pill, paddingHorizontal: 22, paddingVertical: 14 },
  ghost: { borderWidth: 1, borderColor: 'rgba(255,255,255,0.2)', backgroundColor: 'rgba(255,255,255,0.05)', borderRadius: radius.pill, paddingHorizontal: 20, paddingVertical: 14 },
  inkChip: { borderWidth: 1, borderColor: 'rgba(255,255,255,0.15)', backgroundColor: 'rgba(255,255,255,0.05)', borderRadius: radius.pill, paddingHorizontal: 12, paddingVertical: 6 },
});
