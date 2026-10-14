import { Fragment, useEffect, useRef, useState } from 'react';
import { ScrollView, StyleSheet, View } from 'react-native';

import { ResourceCard } from '@/components/lesson/ResourceCard';
import { Chip } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { GradientRing } from '@/components/ui/Decor';
import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT, type TKey } from '@/i18n';
import type { Intent, Session } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useSaveFlashcards, useSendMessage } from '@/lib/queries';
import { colors } from '@/theme/tokens';
import { ChatBubble, TypingBubble } from './ChatBubble';
import { Composer } from './Composer';
import { FlashcardDeck } from './FlashcardDeck';
import { QuizBlock } from './QuizBlock';

const actions: { intent: Intent; label: TKey; icon: 'sparkle' | 'list' | 'book' | 'map' | 'play' | 'target' }[] = [
  { intent: 'simpler', label: 'helper.simpler', icon: 'sparkle' },
  { intent: 'quiz', label: 'helper.quiz', icon: 'list' },
  { intent: 'flashcards', label: 'helper.flashcards', icon: 'book' },
  { intent: 'examples', label: 'helper.examples', icon: 'map' },
  { intent: 'video', label: 'helper.video', icon: 'play' },
  { intent: 'image', label: 'helper.image', icon: 'target' },
];

type Pending = { content?: string; intent?: Intent; label: string };

// The AI helper beside the material: answers questions, and on request makes
// practice quizzes, flashcards and finds real-world videos, links and pictures.
export function HelperPanel({ session, onClose }: { session: Session; onClose?: () => void }) {
  const t = useT();
  const errorText = useErrorText();
  const send = useSendMessage(session.id);
  const saveCards = useSaveFlashcards();
  const [draft, setDraft] = useState('');
  const [pending, setPending] = useState<Pending>();
  const [error, setError] = useState<{ input: Pending; message: string }>();
  const [saved, setSaved] = useState<Record<string, boolean>>({});
  const scrollRef = useRef<ScrollView>(null);
  const msgs = session.messages;

  useEffect(() => {
    const id = setTimeout(() => scrollRef.current?.scrollToEnd({ animated: true }), 60);
    return () => clearTimeout(id);
  }, [msgs.length, pending]);

  const ask = (input: Pending) => {
    setError(undefined);
    setPending(input);
    if (input.content) setDraft('');
    send.mutate(
      { content: input.content, intent: input.intent },
      {
        onSettled: () => setPending(undefined),
        onError: (e) => {
          if (input.content) setDraft(input.content);
          setError({ input, message: errorText(e, 'errors.sendFailed') });
        },
      },
    );
  };

  const save = (messageId: string, cards: { front: string; back: string }[]) =>
    saveCards.mutate({ course: session.courseSlug, nodeId: session.nodeId, cards }, { onSuccess: () => setSaved((s) => ({ ...s, [messageId]: true })) });

  return (
    <View style={styles.panel}>
      <View style={styles.head}>
        <Icon name="sparkle" size={16} color={colors.accent} />
        <Text weight="medium" tight style={{ flex: 1 }}>
          {t('helper.title')}
        </Text>
        {onClose ? <IconButton icon="close" label={t('helper.close')} variant="soft" size={34} onPress={onClose} /> : null}
      </View>

      <ScrollView ref={scrollRef} style={{ flex: 1 }} contentContainerStyle={styles.thread} keyboardShouldPersistTaps="handled">
        <View style={{ flexDirection: 'row', gap: 10 }}>
          <GradientRing size={30} />
          <View style={[styles.greeting, { flex: 1 }]}>
            <Text size={15}>{t('helper.greeting')}</Text>
            {session.greeting ? (
              <Text size={14} color={colors.muted}>
                {session.greeting}
              </Text>
            ) : null}
          </View>
        </View>
        {msgs.length === 0 && session.material?.suggestedQuestions.length && session.canChat ? (
          <View style={{ gap: 8 }}>
            <Eyebrow>{t('helper.suggested')}</Eyebrow>
            {session.material.suggestedQuestions.map((q) => (
              <Chip key={q} label={q} onPress={() => ask({ content: q, label: q })} />
            ))}
          </View>
        ) : null}

        {msgs.map((m, i) => {
          const prev = msgs[i - 1];
          const meta = m.metadata ?? {};
          const searched = m.role === 'assistant' && (meta.intent === 'video' || meta.intent === 'image' || meta.intent === 'examples');
          return (
            <Fragment key={m.id}>
              <ChatBubble message={m} showAvatar={!prev || prev.role !== 'assistant'} />
              {m.role === 'assistant' && meta.quiz?.length ? <QuizBlock questions={meta.quiz} /> : null}
              {m.role === 'assistant' && meta.flashcards?.length ? (
                <FlashcardDeck
                  cards={meta.flashcards}
                  saved={saved[m.id]}
                  saving={saveCards.isPending && saveCards.variables?.cards === meta.flashcards}
                  onSave={() => save(m.id, meta.flashcards!)}
                />
              ) : null}
              {m.role === 'assistant' && meta.resources?.length ? (
                <View style={{ gap: 10 }}>
                  {meta.resources.map((r) => (
                    <ResourceCard key={r.url} item={r} compact />
                  ))}
                </View>
              ) : searched ? (
                <Text size={13} color={colors.muted}>
                  {t('helper.noResults')}
                </Text>
              ) : null}
            </Fragment>
          );
        })}
        {pending ? (
          <>
            <ChatBubble message={{ role: 'user', content: pending.label, metadata: {} }} showAvatar={false} pending />
            <TypingBubble />
          </>
        ) : null}
        {error ? (
          <Card tone="peach" padding={16} style={{ gap: 10 }}>
            <Text size={14}>{error.message}</Text>
            <PillButton label={t('common.retry')} icon="refresh" size="sm" onPress={() => ask(error.input)} />
          </Card>
        ) : null}
      </ScrollView>

      {session.canChat ? (
        <View style={styles.foot}>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={{ gap: 6 }}>
            {actions.map((a) => (
              <Chip key={a.intent} label={t(a.label)} icon={a.icon} onPress={send.isPending ? undefined : () => ask({ intent: a.intent, label: t(a.label) })} />
            ))}
          </ScrollView>
          <Composer value={draft} onChange={setDraft} onSend={(text) => ask({ content: text, label: text })} disabled={send.isPending} />
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  panel: { flex: 1, backgroundColor: colors.card },
  head: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 16, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: colors.line, minHeight: 56 },
  thread: { padding: 16, gap: 14 },
  greeting: { backgroundColor: colors.canvas, borderRadius: 20, borderTopLeftRadius: 6, paddingHorizontal: 16, paddingVertical: 12, gap: 6 },
  foot: { gap: 10, padding: 12, borderTopWidth: 1, borderTopColor: colors.line },
});
