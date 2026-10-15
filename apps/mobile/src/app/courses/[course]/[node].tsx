import { useLocalSearchParams, useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { Chip, DotList, ErrorState, LoadingState, pad2, StatusChip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import { useAuth } from '@/lib/auth';
import { useErrorText } from '@/lib/errors';
import { useNode, useRoadmap } from '@/lib/queries';
import { colors, fonts } from '@/theme/tokens';

export default function NodeScreen() {
  const { course, node: nodeId } = useLocalSearchParams<{ course: string; node: string }>();
  const router = useRouter();
  const authed = useAuth((s) => s.status === 'authenticated');
  const { data: node, isPending, isError, error, refetch } = useNode(course, nodeId);
  // Usually cached from the course page; only used for the goal's label.
  const goal = useRoadmap(course).data?.profile.goal;
  const { wide, gap, contentWidth } = useLayout();
  const t = useT();
  const errorText = useErrorText();

  if (isPending) {
    return (
      <Screen>
        <LoadingState label={t('node.loading')} />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState title={'code' in error && error.code === 'node_not_found' ? t('node.notFound') : t('node.loadError')} message={errorText(error)} onRetry={refetch} />
      </Screen>
    );
  }

  const label = node.status === 'mastered' ? t('node.practiseAgain') : node.openSessionId || node.status === 'in_progress' ? t('node.continueLesson') : t('node.startLesson');
  const mainW = wide ? span(contentWidth, gap, 3, 2) : contentWidth;
  const sideW = wide ? span(contentWidth, gap, 3, 1) : contentWidth;

  const main = (
    <View style={{ width: mainW, gap }}>
      <Card reveal={0} padding={wide ? 32 : 24} style={{ gap: 28 }}>
        <View style={styles.row}>
          <Crumb onPress={() => router.push(`/courses/${node.course.slug}`)} label={node.course.title} />
          <Eyebrow>/ {node.level}</Eyebrow>
        </View>
        <View style={{ gap: 14 }}>
          <StatusChip status={node.status} />
          <Heading size={wide ? 52 : 38}>{node.title}</Heading>
          {node.recommended ? (
            <View style={styles.row}>
              <Icon name="target" size={16} color={colors.accent} strokeWidth={2} />
              <Text size={14} color={colors.accent}>
                {goal ? t('onboarding.relevantFor', { goal: goal.label }) : t('onboarding.recommended')}
              </Text>
            </View>
          ) : null}
          <Text color={colors.muted} size={wide ? 18 : 15} style={{ maxWidth: 520 }}>
            {node.summary}
          </Text>
        </View>
        <View style={{ gap: 10 }}>
          {node.canStart && !authed ? (
            <>
              <PillButton
                label={t('node.logInToStart')}
                icon="arrowUpRight"
                onPress={() => router.push({ pathname: '/login', params: { next: `/courses/${node.course.slug}/${node.id}` } })}
              />
              <Text size={13} color={colors.muted}>
                {t('node.needsAccount')}
              </Text>
            </>
          ) : node.canStart ? (
            <PillButton label={label} icon="arrowUpRight" onPress={() => router.push(`/learn/${node.course.slug}/${node.id}`)} />
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
                  <Chip key={n.id} label={`${n.level} · ${n.title}`} icon={n.status === 'mastered' ? 'check' : n.status === 'locked' ? 'lock' : undefined} onPress={() => router.push(`/courses/${node.course.slug}/${n.id}`)} />
                ))}
              </View>
            )}
          </Card>
        ))}
      </View>
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
                onPress={() => router.push(`/courses/${n.course.slug}/${n.id}`)}
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
      >
        <Eyebrow color="rgba(255,255,255,0.6)">{t('node.mastery')}</Eyebrow>
        <View>
          <View style={{ flexDirection: 'row', alignItems: 'flex-end' }}>
            <Heading size={72} color={colors.white} tabular>
              {node.masteryThreshold}
            </Heading>
            <Heading size={32} color="rgba(255,255,255,0.45)" style={{ marginBottom: 6 }}>
              /100
            </Heading>
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
        <Card reveal={wide ? 2 : 5} style={{ gap: 16 }}>
          <Eyebrow>{t('node.keyIdeas')}</Eyebrow>
          <DotList items={node.keyPoints} />
        </Card>
      ) : null}
    </View>
  );

  return (
    <Screen>
      <View style={{ flexDirection: wide ? 'row' : 'column', gap, alignItems: 'flex-start' }}>
        {main}
        {side}
      </View>
    </Screen>
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
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  objective: { flexDirection: 'row', gap: 12, paddingTop: 2 },
  divider: { borderTopWidth: 1, borderTopColor: colors.line, paddingTop: 18 },
  inkRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'baseline', paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.1)' },
});
