import { StyleSheet, View } from 'react-native';

import { Chip, DotList, pad2, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { GradientRing, MeshBlob } from '@/components/ui/Decor';
import { GradientText } from '@/components/ui/GradientText';
import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { span } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { NodeDetail } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { colors, fonts } from '@/theme/tokens';

// A lesson's details: what it teaches, how to start it, what it needs and
// unlocks. Shown on the lesson page and in the roadmap's lesson dialog.
export function NodeDetailView({
  node,
  width,
  wide,
  gap,
  goal,
  onCourse,
  onNode,
  onGo,
}: {
  node: NodeDetail;
  width: number;
  wide: boolean;
  gap: number;
  /** Label of the learner's goal, for recommended lessons. */
  goal?: string;
  onCourse: () => void;
  /** Another lesson (prerequisite, unlock, foundation). */
  onNode: (course: string, id: string) => void;
  /** Leaves for another page (the lesson, log-in). */
  onGo: (path: string) => void;
}) {
  const t = useT();
  const authed = useAuth((s) => s.status === 'authenticated');
  const contentWidth = width;
  const label = node.status === 'mastered' ? t('node.practiseAgain') : node.openSessionId || node.status === 'in_progress' ? t('node.continueLesson') : t('node.startLesson');
  const mainW = wide ? span(contentWidth, gap, 3, 2) : contentWidth;
  const sideW = wide ? span(contentWidth, gap, 3, 1) : contentWidth;

  const main = (
    <View style={{ width: mainW, gap }}>
      <Card reveal={0} padding={wide ? 32 : 24} style={{ gap: 28 }} decoration={<MeshBlob color={colors.lavender} size={300} style={{ right: -120, top: -140 }} />}>
        <View style={styles.row}>
          <Crumb onPress={onCourse} label={node.course.title} />
          <Eyebrow>/ {node.level}</Eyebrow>
        </View>
        <View style={{ gap: 14 }}>
          <View style={[styles.row, { flexWrap: 'wrap', gap: 8 }]}>
            <StatusChip status={node.status} />
            {node.strand ? <Eyebrow>{t(`node.strand${node.strand}`)}</Eyebrow> : null}
            {node.estimatedMinutes ? (
              <Eyebrow>
                ·{' '}
                {node.estimatedMinutes >= 90
                  ? t('node.hours', { n: Math.round(node.estimatedMinutes / 30) / 2 })
                  : t('node.minutes', { n: node.estimatedMinutes })}
              </Eyebrow>
            ) : null}
          </View>
          <Heading size={wide ? 52 : 38}>{node.title}</Heading>
          {node.recommended ? (
            <View style={styles.row}>
              <Icon name="target" size={16} color={colors.accent} strokeWidth={2} />
              <Text size={14} color={colors.accent}>
                {goal ? t('onboarding.relevantFor', { goal }) : t('onboarding.recommended')}
              </Text>
            </View>
          ) : null}
          <Text color={colors.muted} size={wide ? 18 : 15} style={{ maxWidth: 520 }}>
            {node.summary}
          </Text>
          {node.canDo ? (
            <View style={styles.canDo}>
              <Icon name="check" size={16} color={colors.emeraldDeep} strokeWidth={2.25} />
              <View style={{ flex: 1, gap: 2 }}>
                <Eyebrow color={colors.emeraldDeep}>{t('node.canDo')}</Eyebrow>
                <Text size={15}>{node.canDo}</Text>
              </View>
            </View>
          ) : null}
        </View>
        <View style={{ gap: 10 }}>
          {node.canStart && !authed ? (
            <>
              <PillButton
                label={t('node.logInToStart')}
                icon="arrowUpRight"
                onPress={() => onGo(`/login?next=${encodeURIComponent(`/courses/${node.course.slug}/${node.id}`)}`)}
              />
              <Text size={13} color={colors.muted}>
                {t('node.needsAccount')}
              </Text>
            </>
          ) : node.canStart ? (
            <PillButton label={label} icon="arrowUpRight" onPress={() => onGo(`/learn/${node.course.slug}/${node.id}`)} />
          ) : (
            <>
              <PillButton label={node.status === 'coming_soon' ? t('common.comingSoon') : t('node.lockedButton')} icon={node.status === 'locked' ? 'lock' : undefined} disabled />
              <Text size={13} color={colors.muted}>
                {node.blockedReason}
              </Text>
            </>
          )}
        </View>
      </Card>

      <Card reveal={1} padding={wide ? 32 : 24} style={{ gap: 20 }}>
        <Eyebrow>{t('node.willDo')}</Eyebrow>
        {node.objectives.map((o, i) => (
          <View key={o.id} style={[styles.objective, i > 0 && styles.divider]}>
            <Text size={13} color={colors.accent} style={{ fontFamily: fonts.mono, width: 28, marginTop: 2 }}>
              {pad2(i + 1)}
            </Text>
            <Text size={wide ? 18 : 16} weight="medium" tight style={{ flex: 1 }}>
              {o.text}
            </Text>
          </View>
        ))}
      </Card>

      <View style={{ flexDirection: wide ? 'row' : 'column', gap }}>
        {[
          { title: t('node.prerequisites'), items: node.prerequisites, empty: t('node.noPrereq') },
          { title: t('node.unlocks'), items: node.successors, empty: t('node.noUnlocks') },
        ].map((g, i) => (
          <Card key={g.title} reveal={2 + i} style={{ flex: wide ? 1 : undefined, gap: 14 }}>
            <Eyebrow>{g.title}</Eyebrow>
            {g.items.length === 0 ? (
              <Text size={14} color={colors.muted}>
                {g.empty}
              </Text>
            ) : (
              <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
                {g.items.map((n) => (
                  <Chip key={n.id} label={`${n.level} · ${n.title}`} icon={n.status === 'mastered' ? 'check' : n.status === 'locked' ? 'lock' : undefined} onPress={() => onNode(node.course.slug, n.id)} />
                ))}
              </View>
            )}
          </Card>
        ))}
      </View>
      {node.commonErrors?.length ? (
        <Card reveal={3} style={{ gap: 14 }}>
          <View style={{ gap: 4 }}>
            <Eyebrow>{t('node.commonErrors')}</Eyebrow>
            <Text size={14} color={colors.muted}>
              {t('node.commonErrorsText')}
            </Text>
          </View>
          {node.commonErrors.map((e, i) => (
            <View key={i} style={[styles.error, i > 0 && styles.divider]}>
              <View style={styles.row}>
                <Icon name="close" size={14} color={colors.danger} strokeWidth={2.25} />
                <Text size={15} color={colors.danger} style={{ flex: 1, textDecorationLine: 'line-through' }}>
                  {e.wrong}
                </Text>
              </View>
              <View style={styles.row}>
                <Icon name="check" size={14} color={colors.emeraldDeep} strokeWidth={2.25} />
                <Text size={15} weight="medium" style={{ flex: 1 }}>
                  {e.right}
                </Text>
              </View>
            </View>
          ))}
        </Card>
      ) : null}
      {node.foundations.length ? (
        <Card reveal={4} style={{ gap: 14 }}>
          <View style={{ gap: 4 }}>
            <Eyebrow>{t('node.foundations')}</Eyebrow>
            <Text size={14} color={colors.muted}>
              {t('node.foundationsText', { course: node.foundations[0].course.title })}
            </Text>
          </View>
          <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
            {node.foundations.map((n) => (
              <Chip
                key={`${n.course.slug}/${n.id}`}
                label={`${n.level} · ${n.title}`}
                icon={n.status === 'mastered' ? 'check' : n.status === 'locked' ? 'lock' : undefined}
                onPress={() => onNode(n.course.slug, n.id)}
              />
            ))}
          </View>
        </Card>
      ) : null}
    </View>
  );

  const side = (
    <View style={{ width: sideW, gap }}>
      <Card
        tone="ink"
        reveal={wide ? 1 : 4}
        padding={wide ? 32 : 24}
        style={{ gap: 28 }}
        decoration={<MeshBlob color="rgba(91,76,245,0.5)" size={240} style={{ right: -70, top: -70 }} />}
      >
        <Eyebrow color="rgba(255,255,255,0.6)">{t('node.mastery')}</Eyebrow>
        <View>
          <View style={{ flexDirection: 'row', alignItems: 'flex-end' }}>
            <Heading size={72} color={colors.white} tabular>
              {node.masteryThreshold}
            </Heading>
            <GradientText animated={false} style={{ fontFamily: fonts.semibold, fontSize: 32, lineHeight: 40, letterSpacing: -1.4 }}>
              /100
            </GradientText>
          </View>
          <Text color="rgba(255,255,255,0.7)" style={{ marginTop: 8 }}>
            {t('node.masteryText')}
          </Text>
        </View>
        <View style={{ borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.1)' }}>
          {[
            [t('node.objectives'), node.objectives.length],
            [t('node.keyIdeas'), node.keyPoints.length],
            [t('node.bestScore'), node.masteryScore != null ? String(node.masteryScore) : '—'],
          ].map(([k, v]) => (
            <View key={k} style={styles.inkRow}>
              <Text size={14} color={colors.white}>
                {k}
              </Text>
              <Text size={12} color="rgba(255,255,255,0.5)" style={{ fontFamily: fonts.mono }}>
                {v}
              </Text>
            </View>
          ))}
        </View>
      </Card>
      {node.keyPoints.length > 0 ? (
        <Card tone="lavender" reveal={wide ? 2 : 5} style={{ gap: 16 }}>
          <View style={[styles.row, { justifyContent: 'space-between' }]}>
            <Eyebrow color="rgba(17,17,19,0.6)">{t('node.keyIdeas')}</Eyebrow>
            <GradientRing size={32} />
          </View>
          <DotList items={node.keyPoints} />
        </Card>
      ) : null}
    </View>
  );

  return (
    <View style={{ flexDirection: wide ? 'row' : 'column', gap, alignItems: 'flex-start' }}>
      {main}
      {side}
    </View>
  );
}

function Crumb({ onPress, label }: { onPress: () => void; label: string }) {
  return (
    <Text size={12} color={colors.muted} onPress={onPress} accessibilityRole="link" style={{ fontFamily: fonts.mono, textTransform: 'uppercase', letterSpacing: 0.7, textDecorationLine: 'underline' }}>
      {label}
    </Text>
  );
}

const styles = StyleSheet.create({
  canDo: { flexDirection: 'row', gap: 10, alignItems: 'flex-start', backgroundColor: colors.mint, borderRadius: 16, padding: 14, maxWidth: 560 },
  error: { gap: 6 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  objective: { flexDirection: 'row', gap: 12, paddingTop: 2 },
  divider: { borderTopWidth: 1, borderTopColor: colors.line, paddingTop: 18 },
  inkRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'baseline', paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.1)' },
});
