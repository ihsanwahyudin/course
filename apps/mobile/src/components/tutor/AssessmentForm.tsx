import { Platform, StyleSheet, TextInput, View } from 'react-native';

import { pad2 } from '@/components/ui/Bits';
import { useT } from '@/i18n';
import { Card } from '@/components/ui/Card';
import { MeshBlob } from '@/components/ui/Decor';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import type { Session } from '@/lib/api';
import { useUI } from '@/lib/store';
import { colors, fonts, radius } from '@/theme/tokens';

export function AssessmentForm({ session, onSubmit, submitting, error }: { session: Session; onSubmit: (answers: { itemId: string; answer: string }[]) => void; submitting: boolean; error?: string }) {
  const t = useT();
  const drafts = useUI((s) => s.drafts[session.id]) ?? {};
  const setDraft = useUI((s) => s.setDraft);
  const items = session.assessment;
  const answered = items.filter((it) => (drafts[it.id] ?? '').trim()).length;
  const objective = (id: string) => session.node.objectives.find((o) => o.id === id)?.text;

  return (
    <Card padding={24} style={{ gap: 22 }} decoration={<MeshBlob color={colors.lavender} size={260} style={{ right: -110, top: -120 }} />}>
      <View style={{ gap: 10 }}>
        <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
          <Eyebrow color={colors.accent}>{t('assessment.title')}</Eyebrow>
          <Eyebrow>{t('assessment.answered', { done: pad2(answered), total: pad2(items.length) })}</Eyebrow>
        </View>
        <Heading size={30}>{t('assessment.heading')}</Heading>
        <Text color={colors.muted}>{t('assessment.intro', { threshold: session.node.masteryThreshold, title: session.node.title })}</Text>
      </View>
      {items.map((it, i) => (
        <View key={it.id} style={[{ gap: 10 }, i > 0 && styles.divider]}>
          <View style={{ flexDirection: 'row', gap: 12 }}>
            <Text size={13} color={colors.accent} style={{ fontFamily: fonts.mono, marginTop: 2 }}>
              {pad2(i + 1)}
            </Text>
            <View style={{ flex: 1, gap: 4 }}>
              <Text weight="medium" tight size={16}>
                {it.prompt}
              </Text>
              {objective(it.objectiveId) ? <Eyebrow>{objective(it.objectiveId)}</Eyebrow> : null}
            </View>
          </View>
          <TextInput
            value={drafts[it.id] ?? ''}
            onChangeText={(v) => setDraft(session.id, it.id, v)}
            editable={!submitting}
            placeholder={t('assessment.yourAnswer')}
            placeholderTextColor={colors.muted}
            accessibilityLabel={t('assessment.answerLabel', { n: i + 1, prompt: it.prompt })}
            maxLength={1000}
            multiline
            style={styles.input}
          />
        </View>
      ))}
      {error ? (
        <View style={styles.error}>
          <Text size={14}>{error}</Text>
        </View>
      ) : null}
      <PillButton
        label={submitting ? t('assessment.scoring') : t('assessment.submit')}
        icon="arrowUpRight"
        loading={submitting}
        disabled={answered < items.length}
        onPress={() => onSubmit(items.map((it) => ({ itemId: it.id, answer: (drafts[it.id] ?? '').trim() })))}
        accessibilityHint={t('assessment.hint')}
      />
    </Card>
  );
}

const styles = StyleSheet.create({
  divider: { borderTopWidth: 1, borderTopColor: colors.line, paddingTop: 20 },
  input: {
    backgroundColor: colors.canvas,
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: radius.xxl,
    paddingHorizontal: 14,
    paddingVertical: 12,
    minHeight: 48,
    fontFamily: fonts.regular,
    fontSize: 15,
    color: colors.ink,
    ...(Platform.OS === 'web' ? ({ outlineColor: colors.accent } as object) : null),
  },
  error: { backgroundColor: colors.peach, borderRadius: radius.xxl, padding: 14 },
});
