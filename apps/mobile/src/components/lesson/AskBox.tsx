import { useState } from 'react';
import { ActivityIndicator, Platform, Pressable, StyleSheet, TextInput, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import { ApiError, type AskEntry } from '@/lib/api';
import { useRecorder } from '@/lib/audio';
import { useErrorText } from '@/lib/errors';
import { colors, fonts, radius } from '@/theme/tokens';
import { RichText } from './RichText';

// Mic button: tap to record, tap again to stop; the text lands in the input.
export function MicButton({ onText, size = 36 }: { onText: (text: string) => void; size?: number }) {
  const t = useT();
  const errorText = useErrorText();
  const rec = useRecorder(onText);
  const label = rec.recording ? t('step.stopRecording') : t('step.speak');
  return (
    <View style={{ alignItems: 'flex-end' }}>
      <Pressable
        onPress={() => (rec.recording ? rec.stop() : rec.start())}
        disabled={rec.busy}
        accessibilityRole="button"
        accessibilityLabel={label}
        hitSlop={4}
        style={[styles.mic, { width: size, height: size, borderRadius: size / 2 }, rec.recording && { backgroundColor: colors.danger }]}
      >
        {rec.busy ? <ActivityIndicator size="small" color={colors.accent} /> : <Icon name={rec.recording ? 'stop' : 'mic'} size={Math.round(size * 0.48)} color={rec.recording ? colors.white : colors.ink} />}
      </Pressable>
      {rec.recording || rec.busy || rec.error ? (
        <View style={styles.micNote} accessibilityLiveRegion="polite">
          <Text size={12} color={rec.error ? colors.danger : colors.muted}>
            {rec.error
              ? errorText(new ApiError(0, rec.error, '', false))
              : rec.busy
                ? t('step.transcribing')
                : t('step.recording', { sec: Math.floor(rec.durationMs / 1000) })}
          </Text>
        </View>
      ) : null}
    </View>
  );
}

function Answer({ a }: { a: AskEntry }) {
  const t = useT();
  return (
    <View style={{ gap: 12 }}>
      <View style={styles.q}>
        <Text size={15} color={colors.white}>
          {a.question}
        </Text>
      </View>
      <View style={styles.a}>
        <Icon name="sparkle" size={16} color={colors.accent} />
        <View style={{ flex: 1, gap: 6 }}>
          {a.scope !== 'in' ? (
            <View style={styles.scope}>
              <Text size={12} color={colors.muted}>
                {a.scope === 'other_lesson' && a.otherLesson ? t('step.otherLesson', { title: a.otherLesson }) : t('step.offTopic')}
              </Text>
            </View>
          ) : null}
          <RichText size={15}>{a.answer}</RichText>
        </View>
      </View>
    </View>
  );
}

const CHAT_PAGE_SIZE = 5;

// Every step owns its collapsed state and visible history independently.
export function AskBox({ asks, onAsk, pending = null, error }: { asks: AskEntry[]; onAsk?: (q: string) => void; pending?: string | null; error?: string }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [visibleCount, setVisibleCount] = useState(CHAT_PAGE_SIZE);
  const [draft, setDraft] = useState('');
  const firstVisible = Math.max(0, asks.length - visibleCount);
  const send = () => {
    const q = draft.trim();
    if (!q || pending || !onAsk) return;
    onAsk(q);
    setDraft('');
  };
  return (
    <View style={styles.box}>
      <Pressable
        accessibilityRole="button"
        accessibilityState={{ expanded: open }}
        onPress={() => {
          setOpen((v) => !v);
          setVisibleCount(CHAT_PAGE_SIZE);
        }}
        style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}
      >
        <Icon name="chat" size={15} color={colors.accent} />
        <Eyebrow color={colors.accent} style={{ flex: 1 }}>{t('step.askTitle')}</Eyebrow>
        {asks.length > 0 ? <Text size={12} color={colors.muted}>{asks.length}</Text> : null}
        <View style={{ transform: [{ rotate: open ? '90deg' : '0deg' }] }}>
          <Icon name="chevronRight" size={16} color={colors.accent} />
        </View>
      </Pressable>
      {open ? (
        <View style={{ gap: 16 }}>
          {firstVisible > 0 ? (
            <PillButton label={t('step.loadEarlier')} variant="secondary" onPress={() => setVisibleCount((n) => n + CHAT_PAGE_SIZE)} />
          ) : null}
          {asks.slice(firstVisible).map((a, i) => (
            <Answer key={firstVisible + i} a={a} />
          ))}
          {pending ? (
            <View style={{ gap: 12 }}>
              <View style={styles.q}><Text size={15} color={colors.white}>{pending}</Text></View>
              <View style={styles.a} accessibilityLiveRegion="polite">
                <ActivityIndicator size="small" color={colors.accent} />
                <Text size={14} color={colors.muted}>{t('step.thinking')}</Text>
              </View>
            </View>
          ) : null}
          {error ? <Text size={14} color={colors.danger}>{error}</Text> : null}
          {onAsk ? (
            <View style={styles.inputRow}>
              <TextInput
                value={draft}
                onChangeText={setDraft}
                onSubmitEditing={send}
                placeholder={t('step.askPlaceholder')}
                placeholderTextColor={colors.muted}
                accessibilityLabel={t('step.askLabel')}
                maxLength={2000}
                editable={!pending}
                returnKeyType="send"
                style={styles.input}
              />
              <MicButton onText={(txt) => setDraft((d) => (d ? `${d} ${txt}` : txt))} />
              <IconButton icon="arrowUp" label={t('step.send')} variant="accent" size={36} onPress={pending || !draft.trim() ? undefined : send} />
            </View>
          ) : null}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { backgroundColor: colors.accentSoft, borderRadius: 24, padding: 16, gap: 16 },
  q: { alignSelf: 'flex-end', maxWidth: '88%', backgroundColor: colors.ink, borderRadius: 18, borderBottomRightRadius: 6, paddingHorizontal: 16, paddingVertical: 11 },
  a: { flexDirection: 'row', gap: 12, alignItems: 'flex-start', paddingHorizontal: 4 },
  scope: { alignSelf: 'flex-start', backgroundColor: colors.card, borderRadius: radius.pill, paddingHorizontal: 10, paddingVertical: 3 },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: colors.card,
    borderWidth: 1.5,
    borderColor: colors.accent,
    borderRadius: radius.pill,
    paddingLeft: 16,
    paddingRight: 6,
    paddingVertical: 5,
  },
  input: {
    flex: 1,
    minHeight: 36,
    fontFamily: fonts.regular,
    fontSize: 15,
    color: colors.ink,
    ...(Platform.OS === 'web' ? ({ outlineStyle: 'none' } as object) : null),
  },
  mic: { backgroundColor: colors.inkSoft, alignItems: 'center', justifyContent: 'center' },
  micNote: { position: 'absolute', top: '100%', right: 0, marginTop: 6, backgroundColor: colors.card, borderRadius: radius.lg, paddingHorizontal: 8, paddingVertical: 4, minWidth: 180, boxShadow: '0px 4px 16px rgba(17,17,19,0.12)' },
});
