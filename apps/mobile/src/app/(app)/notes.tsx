import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { ArtifactView } from '@/components/lesson/Widgets';
import { PageHeader } from '@/components/PageHeader';
import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Screen, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useFormatDate, useT } from '@/i18n';
import type { Note } from '@/lib/api';
import { confirmAction } from '@/lib/confirm';
import { useErrorText } from '@/lib/errors';
import { useDeleteNote, useNotes } from '@/lib/queries';
import { colors } from '@/theme/tokens';

function NoteCard({ note, index }: { note: Note; index: number }) {
  const t = useT();
  const router = useRouter();
  const formatDate = useFormatDate();
  const del = useDeleteNote();
  const [open, setOpen] = useState(index === 0);
  return (
    <Card reveal={index} spotlight={false} style={{ gap: 16 }}>
      <Pressable onPress={() => setOpen((o) => !o)} accessibilityRole="button" accessibilityState={{ expanded: open }} style={styles.head}>
        <View style={styles.icon}>
          <Icon name="book" size={18} color={colors.accent} />
        </View>
        <View style={{ flex: 1, gap: 2 }}>
          <Eyebrow numberOfLines={1}>{`${note.courseTitle} · ${note.nodeTitle}`}</Eyebrow>
          <Text size={17} weight="medium" tight>
            {note.title}
          </Text>
          <Text size={12} color={colors.muted}>
            {t('notes.saved', { date: formatDate(note.createdAt) })}
          </Text>
        </View>
        <View style={{ transform: [{ rotate: open ? '90deg' : '0deg' }] }}>
          <Icon name="chevronRight" size={18} color={colors.muted} />
        </View>
      </Pressable>
      {open ? (
        <>
          <ArtifactView artifact={note.content} />
          <View style={{ flexDirection: 'row', gap: 8, alignItems: 'center' }}>
            <PillButton label={t('notes.openLesson')} icon="arrowUpRight" variant="secondary" size="sm" onPress={() => router.push(`/learn/${note.courseSlug}/${note.nodeId}`)} />
            <View style={{ flex: 1 }} />
            <IconButton
              icon="trash"
              label={t('notes.delete')}
              variant="soft"
              size={36}
              onPress={async () => {
                if (await confirmAction(`${t('notes.delete')}: ${note.title}?`, t('memory.delete'), t('memory.cancel'))) del.mutate(note.id);
              }}
            />
          </View>
        </>
      ) : null}
    </Card>
  );
}

// "Catatanku": cheat sheets saved from lessons.
export default function NotesScreen() {
  const t = useT();
  const router = useRouter();
  const errorText = useErrorText();
  const { wide } = useLayout();
  const { data, isPending, isError, error, refetch } = useNotes();
  return (
    <Screen maxWidth={860}>
      <PageHeader label={t('notes.title')} title={t('notes.title')} description={t('notes.description')} />
      {isPending ? (
        <LoadingState label={t('notes.loading')} />
      ) : isError ? (
        <ErrorState title={t('notes.loadError')} message={errorText(error)} onRetry={refetch} />
      ) : data.length === 0 ? (
        <Card style={{ gap: 12 }}>
          <Heading size={26}>{t('notes.empty')}</Heading>
          <Text color={colors.muted}>{t('notes.emptyText')}</Text>
          <PillButton label={t('common.browseCourses')} icon="arrowUpRight" onPress={() => router.push('/courses')} />
        </Card>
      ) : (
        <View style={{ gap: wide ? 16 : 12 }}>
          {data.map((n, i) => (
            <NoteCard key={n.id} note={n} index={i} />
          ))}
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  head: { flexDirection: 'row', alignItems: 'center', gap: 14 },
  icon: { width: 44, height: 44, borderRadius: 12, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
});
