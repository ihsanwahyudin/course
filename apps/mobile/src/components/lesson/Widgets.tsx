import { Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Artifact, Pair, StepView, Timeline } from '@/lib/api';
import { useSpeak } from '@/lib/audio';
import { colors, radius } from '@/theme/tokens';
import { ResourceCard } from './ResourceCard';
import { RichText } from './RichText';

// White widget card used by every step.
export function WidgetCard({ eyebrow, title, children, tone }: { eyebrow?: string; title?: string; children?: React.ReactNode; tone?: 'accent' }) {
  return (
    <View style={[styles.card, tone === 'accent' && { borderColor: colors.accent }]}>
      {eyebrow ? <Eyebrow color={colors.accent}>{eyebrow}</Eyebrow> : null}
      {title ? <Heading size={22}>{title}</Heading> : null}
      {children}
    </View>
  );
}

type Speech = ReturnType<typeof useSpeak>;

// Which item of a list is being read: by itself, or as part of "listen to all".
function playingIndex(sp: Speech, texts: string[]) {
  if (sp.speaking === null) return -1;
  if (sp.speaking === texts.join('\n') && texts.length > 1) return sp.index ?? -1;
  return texts.indexOf(sp.speaking);
}

// Round speaker button for one English sentence.
function SpeakButton({ text, on, onPress, size = 32 }: { text: string; on: boolean; onPress: () => void; size?: number }) {
  const t = useT();
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={on ? t('step.stop') : t('step.listen', { text })}
      hitSlop={6}
      style={[styles.speak, { width: size, height: size, borderRadius: size / 2 }, on && { backgroundColor: colors.accent }]}
    >
      <Icon name={on ? 'stop' : 'volume'} size={Math.round(size * 0.5)} color={on ? colors.white : colors.accent} />
    </Pressable>
  );
}

function ListenAll({ texts, sp }: { texts: string[]; sp: Speech }) {
  const t = useT();
  const on = texts.length > 1 && sp.speaking === texts.join('\n');
  if (texts.length < 2) return null;
  return (
    <Pressable onPress={() => (on ? sp.stop() : sp.speakAll(texts))} accessibilityRole="button" style={styles.listenAll}>
      <Icon name={on ? 'stop' : 'volume'} size={16} color={colors.accent} />
      <Text size={14} weight="medium" color={colors.accent}>
        {on ? t('step.stop') : t('step.listenAll')}
      </Text>
    </Pressable>
  );
}

// Kartu konsep: explanation plus word tiles (English over its meaning). Tap a
// tile to hear it; "listen to all" reads them in turn and highlights each.
export function ConceptCard({ step }: { step: StepView }) {
  const t = useT();
  const sp = useSpeak();
  const pairs = step.pairs ?? [];
  const texts = pairs.map((p) => p.en);
  const playing = playingIndex(sp, texts);
  return (
    <WidgetCard eyebrow={t('step.conceptCard')} title={step.title}>
      {step.body ? <RichText size={15} color={colors.muted}>{step.body}</RichText> : null}
      {pairs.length ? (
        <View style={styles.tiles}>
          {pairs.map((p, i) => {
            const on = i === playing;
            return (
              <Pressable
                key={i}
                onPress={() => (on ? sp.stop() : sp.speak(p.en))}
                accessibilityRole="button"
                accessibilityLabel={on ? t('step.stop') : t('step.listen', { text: p.en })}
                accessibilityState={{ selected: on }}
                style={[styles.tile, on && styles.tileOn]}
              >
                <Text size={16} weight="semibold" tight color={on ? colors.accent : colors.ink}>
                  {p.en}
                </Text>
                <Text size={12} color={on ? colors.accent : colors.muted}>
                  {p.id}
                </Text>
              </Pressable>
            );
          })}
        </View>
      ) : null}
      <ListenAll texts={texts} sp={sp} />
    </WidgetCard>
  );
}

// Example sentences with audio; the row being read is highlighted.
export function ExampleList({ examples }: { examples: Pair[] }) {
  const sp = useSpeak();
  const texts = examples.map((e) => e.en);
  const playing = playingIndex(sp, texts);
  return (
    <View style={{ gap: 4 }}>
      {examples.map((e, i) => {
        const on = i === playing;
        return (
          <View key={i} style={[styles.example, i > 0 && { borderTopColor: colors.line }, on && styles.exampleOn]}>
            <SpeakButton text={e.en} on={on} onPress={() => (on ? sp.stop() : sp.speak(e.en))} />
            <View style={{ flex: 1, gap: 1 }}>
              <Text size={16} weight="medium" tight selectable color={on ? colors.accent : colors.ink}>
                {e.en}
              </Text>
              <Text size={13} color={colors.muted} selectable>
                {e.id}
              </Text>
            </View>
          </View>
        );
      })}
      <ListenAll texts={texts} sp={sp} />
    </View>
  );
}

export function ExamplesCard({ step }: { step: StepView }) {
  return (
    <WidgetCard eyebrow={step.title}>
      {step.body ? <RichText size={15} color={colors.muted}>{step.body}</RichText> : null}
      <ExampleList examples={step.examples ?? []} />
    </WidgetCard>
  );
}

// Tense timeline: past · now · future marks on one line.
export function TimelineCard({ step, timeline }: { step: StepView; timeline: Timeline }) {
  const tint = { past: colors.muted, now: colors.accent, future: colors.emeraldDeep } as const;
  return (
    <WidgetCard eyebrow={step.title} title={timeline.title}>
      {step.body ? <RichText size={15} color={colors.muted}>{step.body}</RichText> : null}
      <View style={styles.timeline} accessibilityRole="list">
        <View style={styles.timelineLine} />
        {timeline.marks.map((m, i) => (
          <View key={i} style={styles.mark} accessibilityRole="text">
            <View style={[styles.markDot, { backgroundColor: tint[m.kind] ?? colors.muted }, m.kind === 'now' && styles.markNow]} />
            <Text size={13} weight={m.kind === 'now' ? 'medium' : 'regular'} color={m.kind === 'now' ? colors.ink : colors.muted} style={{ textAlign: 'center' }}>
              {m.label}
            </Text>
          </View>
        ))}
      </View>
      {timeline.caption ? (
        <Text size={14} color={colors.muted}>
          {timeline.caption}
        </Text>
      ) : null}
    </WidgetCard>
  );
}

// Inline card that opens the cheat sheet in the side panel / sheet.
export function ArtifactPreview({ artifact, onOpen }: { artifact: Artifact; onOpen: () => void }) {
  const t = useT();
  return (
    <Pressable onPress={onOpen} accessibilityRole="button" accessibilityLabel={`${t('step.open')}: ${artifact.title}`} style={styles.artifact}>
      <View style={styles.artifactIcon}>
        <Icon name="panel" size={18} color={colors.accent} />
      </View>
      <View style={{ flex: 1, gap: 2 }}>
        <Text weight="medium" size={15} tight numberOfLines={2}>
          {artifact.title}
        </Text>
        <Text size={13} color={colors.muted}>
          {t('step.artifactHint')}
        </Text>
      </View>
      <View style={styles.openPill}>
        <Text size={13} weight="medium">
          {t('step.open')}
        </Text>
      </View>
    </Pressable>
  );
}

// The full cheat sheet: table, tips and examples.
export function ArtifactView({ artifact }: { artifact: Artifact }) {
  const t = useT();
  const cols = artifact.headers.length || (artifact.rows[0]?.length ?? 0);
  const cellColor = (j: number) => (j === 0 ? colors.ink : cols >= 3 && j < cols - 1 ? colors.accent : colors.muted);
  return (
    <View style={{ gap: 18 }}>
      {artifact.intro ? <RichText size={15} color={colors.muted}>{artifact.intro}</RichText> : null}
      {artifact.rows.length ? (
        <View style={styles.table}>
          {artifact.headers.length ? (
            <View style={[styles.tr, { backgroundColor: colors.canvas }]}>
              {artifact.headers.map((h, j) => (
                <Eyebrow key={j} style={styles.td}>
                  {h}
                </Eyebrow>
              ))}
            </View>
          ) : null}
          {artifact.rows.map((r, i) => (
            <View key={i} style={[styles.tr, i % 2 === 1 && { backgroundColor: 'rgba(17,17,19,0.025)' }]}>
              {r.map((c, j) => (
                <Text key={j} size={j === cols - 1 && cols > 1 ? 13 : 15} weight={j < cols - 1 || cols === 1 ? 'medium' : 'regular'} color={cellColor(j)} style={styles.td} selectable>
                  {c}
                </Text>
              ))}
            </View>
          ))}
        </View>
      ) : null}
      {artifact.tips.map((tip, i) => (
        <View key={i} style={styles.tip}>
          <Icon name="sparkle" size={16} color={colors.accent} />
          <View style={{ flex: 1 }}>
            <RichText size={14}>{tip}</RichText>
          </View>
        </View>
      ))}
      {artifact.examples.length ? (
        <View style={{ gap: 8 }}>
          <Eyebrow>{t('step.examplesAt')}</Eyebrow>
          <ExampleList examples={artifact.examples} />
        </View>
      ) : null}
    </View>
  );
}

export function ResourceList({ step }: { step: StepView }) {
  return (
    <View style={{ gap: 12 }}>
      {step.body ? <RichText size={15} color={colors.muted}>{step.body}</RichText> : null}
      {(step.resources ?? []).map((r) => (
        <ResourceCard key={r.url} item={r} />
      ))}
    </View>
  );
}

// "Simpan ke Catatanku" + "Buat flashcard" under a cheat sheet.
export function ArtifactActions({ onSaveNote, onMakeCards, noteSaved, cardsMade, saving, making }: {
  onSaveNote: () => void;
  onMakeCards: () => void;
  noteSaved: boolean;
  cardsMade: boolean;
  saving: boolean;
  making: boolean;
}) {
  const t = useT();
  return (
    <View style={{ flexDirection: 'row', gap: 10, flexWrap: 'wrap' }}>
      <PillButton label={noteSaved ? t('step.noteSaved') : t('step.saveNote')} icon={noteSaved ? 'check' : undefined} disabled={noteSaved} loading={saving} onPress={onSaveNote} />
      <PillButton label={cardsMade ? t('step.cardsMade') : t('step.makeCards')} icon={cardsMade ? 'check' : undefined} variant="secondary" disabled={cardsMade} loading={making} onPress={onMakeCards} />
    </View>
  );
}

// Flashcards from a cheat sheet: its examples, else its rows.
export function artifactCards(a: Artifact) {
  if (a.examples.length) return a.examples.map((e) => ({ front: e.en, back: e.id }));
  return a.rows.filter((r) => r.length >= 2).map((r) => ({ front: r.slice(0, -1).join(' · '), back: r[r.length - 1] }));
}

const styles = StyleSheet.create({
  card: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: 24, padding: 20, gap: 14 },
  tiles: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  tile: { backgroundColor: colors.canvas, borderWidth: 1.5, borderColor: 'transparent', borderRadius: radius.xl, paddingHorizontal: 14, paddingVertical: 10, gap: 2, minWidth: 56 },
  tileOn: { backgroundColor: colors.accentSoft, borderColor: colors.accent },
  listenAll: { flexDirection: 'row', alignItems: 'center', gap: 8, alignSelf: 'flex-start', paddingVertical: 4 },
  speak: { backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
  example: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 10, paddingHorizontal: 8, marginHorizontal: -8, borderRadius: radius.xl, borderTopWidth: 1, borderTopColor: 'transparent' },
  exampleOn: { backgroundColor: colors.accentSoft },
  timeline: { flexDirection: 'row', justifyContent: 'space-between', paddingTop: 8, paddingBottom: 4 },
  timelineLine: { position: 'absolute', left: 16, right: 16, top: 15, height: 2, backgroundColor: colors.line },
  mark: { flex: 1, alignItems: 'center', gap: 8 },
  markDot: { width: 14, height: 14, borderRadius: 7, borderWidth: 3, borderColor: colors.card },
  markNow: { width: 18, height: 18, borderRadius: 9, marginTop: -2 },
  artifact: { flexDirection: 'row', alignItems: 'center', gap: 14, backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: 20, padding: 14 },
  artifactIcon: { width: 44, height: 44, borderRadius: radius.xl, backgroundColor: colors.accentSoft, alignItems: 'center', justifyContent: 'center' },
  openPill: { borderWidth: 1, borderColor: colors.line, borderRadius: radius.pill, paddingHorizontal: 12, paddingVertical: 6 },
  table: { borderWidth: 1, borderColor: colors.line, borderRadius: radius.xxl, overflow: 'hidden', backgroundColor: colors.card },
  tr: { flexDirection: 'row', paddingHorizontal: 14, paddingVertical: 11, gap: 10 },
  td: { flex: 1 },
  tip: { flexDirection: 'row', gap: 10, backgroundColor: colors.accentSoft, borderRadius: radius.xxl, padding: 14 },
});
