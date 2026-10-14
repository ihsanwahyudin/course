import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Platform, StyleSheet, Switch, TextInput, View } from 'react-native';

import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useFormatDate, useT } from '@/i18n';
import type { MemoryItem } from '@/lib/api';
import { confirmAction } from '@/lib/confirm';
import { useErrorText } from '@/lib/errors';
import { useMemory, useMemoryAction } from '@/lib/queries';
import { colors, fonts, radius } from '@/theme/tokens';

function Group({ title, source, children }: { title: string; source: string; children: React.ReactNode }) {
  return (
    <View style={{ gap: 4 }}>
      <View style={styles.groupHead}>
        <Eyebrow>{title}</Eyebrow>
        <Eyebrow style={{ textTransform: 'none' }}>{source}</Eyebrow>
      </View>
      {children}
    </View>
  );
}

function Item({ item }: { item: MemoryItem }) {
  const t = useT();
  const formatDate = useFormatDate();
  const act = useMemoryAction();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(item.text);
  const meta = [item.sourceTitle, formatDate(item.createdAt, { day: 'numeric', month: 'short' })].filter(Boolean).join(' · ');
  if (editing) {
    return (
      <View style={[styles.item, { flexDirection: 'column', alignItems: 'stretch', gap: 10 }]}>
        <TextInput value={draft} onChangeText={setDraft} maxLength={200} autoFocus accessibilityLabel={t('memory.editLabel')} style={styles.input} />
        <View style={{ flexDirection: 'row', gap: 8, justifyContent: 'flex-end' }}>
          <PillButton label={t('memory.cancel')} variant="secondary" size="sm" onPress={() => (setEditing(false), setDraft(item.text))} />
          <PillButton
            label={t('memory.save')}
            size="sm"
            loading={act.isPending}
            disabled={!draft.trim()}
            onPress={() => act.mutate({ kind: 'edit', id: item.id, text: draft }, { onSuccess: () => setEditing(false) })}
          />
        </View>
      </View>
    );
  }
  return (
    <View style={styles.item}>
      <View style={{ flex: 1, gap: 2 }}>
        <Text size={15}>{item.text}</Text>
        <Text size={12} color={colors.muted}>
          {meta}
        </Text>
      </View>
      <IconButton icon="edit" label={t('memory.edit')} variant="soft" size={32} onPress={() => setEditing(true)} />
      <IconButton icon="trash" label={t('memory.delete')} variant="soft" size={32} onPress={() => act.mutate({ kind: 'delete', id: item.id })} />
    </View>
  );
}

// Account → "What the tutor remembers about you": onboarding answers
// (edited through onboarding), learning patterns and personal notes, a
// switch for new memories, and delete-all.
export function MemorySection({ reveal }: { reveal?: number }) {
  const t = useT();
  const router = useRouter();
  const errorText = useErrorText();
  const { data, isPending, isError, error, refetch } = useMemory();
  const act = useMemoryAction();
  const patterns = data?.items.filter((i) => i.kind === 'pattern') ?? [];
  const personal = data?.items.filter((i) => i.kind === 'personal') ?? [];
  return (
    <Card reveal={reveal} spotlight={false} style={{ gap: 18 }}>
      <View style={{ flexDirection: 'row', gap: 14, alignItems: 'flex-start' }}>
        <View style={styles.icon}>
          <Icon name="brain" size={20} color={colors.accent} />
        </View>
        <View style={{ flex: 1, gap: 4 }}>
          <Text size={18} weight="medium" tight>
            {t('memory.title')}
          </Text>
          <Text size={14} color={colors.muted}>
            {t('memory.text')}
          </Text>
        </View>
      </View>
      {isPending ? (
        <LoadingState label={t('memory.loading')} />
      ) : isError ? (
        <ErrorState message={errorText(error)} onRetry={refetch} />
      ) : (
        <>
          <View style={styles.toggle}>
            <Text size={14} style={{ flex: 1 }}>
              {t('memory.toggle')}
            </Text>
            <Switch
              value={data.enabled}
              onValueChange={(enabled) => act.mutate({ kind: 'toggle', enabled })}
              trackColor={{ true: colors.accent, false: colors.lineStrong }}
              thumbColor={colors.white}
              {...(Platform.OS === 'web' ? ({ activeThumbColor: colors.white } as object) : null)}
              accessibilityLabel={t('memory.toggle')}
            />
          </View>
          {data.profile.length ? (
            <Group title={t('memory.profile')} source={t('memory.fromOnboarding')}>
              {data.profile.map((p, i) => (
                <View key={i} style={styles.item}>
                  <View style={{ flex: 1, gap: 2 }}>
                    <Text size={15}>{p.answer}</Text>
                    <Text size={12} color={colors.muted}>{`${p.question} · ${p.courseTitle}`}</Text>
                  </View>
                  <IconButton icon="edit" label={t('memory.editProfile')} variant="soft" size={32} onPress={() => router.push(`/onboarding/${p.course}`)} />
                </View>
              ))}
            </Group>
          ) : null}
          <Group title={t('memory.patterns')} source={t('memory.fromLessons')}>
            {patterns.length ? patterns.map((m) => <Item key={m.id} item={m} />) : <Text size={14} color={colors.muted} style={styles.empty}>{t('memory.empty')}</Text>}
          </Group>
          <Group title={t('memory.personal')} source={t('memory.fromQuestions')}>
            {personal.length ? personal.map((m) => <Item key={m.id} item={m} />) : <Text size={14} color={colors.muted} style={styles.empty}>{t('memory.empty')}</Text>}
          </Group>
          {act.error ? (
            <Text size={14} color={colors.danger}>
              {errorText(act.error)}
            </Text>
          ) : null}
          <View style={styles.foot}>
            <Text size={13} color={colors.muted} style={{ flex: 1, minWidth: 200 }}>
              {t('memory.privacy')}
            </Text>
            {data.items.length ? (
              <PillButton
                label={t('memory.deleteAll')}
                variant="secondary"
                size="sm"
                style={{ borderColor: colors.danger }}
                onPress={async () => {
                  if (await confirmAction(t('memory.confirmDeleteAll'), t('memory.confirm'), t('memory.cancel'))) act.mutate({ kind: 'deleteAll' });
                }}
              />
            ) : null}
          </View>
        </>
      )}
    </Card>
  );
}

const styles = StyleSheet.create({
  icon: { width: 44, height: 44, borderRadius: 12, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
  toggle: { flexDirection: 'row', alignItems: 'center', gap: 12, backgroundColor: colors.canvas, borderRadius: radius.xxl, paddingHorizontal: 16, paddingVertical: 12 },
  groupHead: { flexDirection: 'row', justifyContent: 'space-between', gap: 12, paddingTop: 6, paddingBottom: 4 },
  item: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingVertical: 10, borderBottomWidth: 1, borderBottomColor: colors.line },
  empty: { paddingVertical: 8 },
  input: {
    backgroundColor: colors.canvas,
    borderWidth: 1,
    borderColor: colors.accent,
    borderRadius: radius.xl,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontFamily: fonts.regular,
    fontSize: 15,
    color: colors.ink,
    ...(Platform.OS === 'web' ? ({ outlineStyle: 'none' } as object) : null),
  },
  foot: { flexDirection: 'row', alignItems: 'center', gap: 12, flexWrap: 'wrap', paddingTop: 4 },
});
