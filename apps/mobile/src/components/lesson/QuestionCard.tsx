import { useState } from 'react';
import { Platform, Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { AnswerInput, QuestionView } from '@/lib/api';
import { colors, radius } from '@/theme/tokens';
import { ActionRow, LinkButton } from './StepParts';
import { RichText } from './RichText';

export type QuestionBusy = 'answer' | 'hint' | 'reveal' | null;

function Option({ label, state, onPress, disabled }: { label: string; state: 'idle' | 'picked' | 'right' | 'wrong' | 'dim'; onPress: () => void; disabled: boolean }) {
  const [hovered, setHovered] = useState(false);
  const look = {
    idle: { bg: hovered && !disabled ? colors.canvas : colors.card, border: colors.lineStrong, fg: colors.ink },
    picked: { bg: colors.accentSoft, border: colors.accent, fg: colors.accent },
    right: { bg: colors.mint, border: colors.emerald, fg: colors.emeraldDeep },
    wrong: { bg: colors.peach, border: colors.danger, fg: colors.danger },
    dim: { bg: colors.card, border: colors.line, fg: colors.muted },
  }[state];
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="radio"
      accessibilityState={{ selected: state === 'picked' || state === 'right', disabled }}
      style={[styles.option, { backgroundColor: look.bg, borderColor: look.border }, Platform.OS === 'web' && ({ cursor: disabled ? 'default' : 'pointer' } as object)]}
    >
      {state === 'right' ? <Icon name="check" size={14} color={look.fg} strokeWidth={2.5} /> : null}
      <Text size={16} weight="medium" color={look.fg}>
        {label}
      </Text>
    </Pressable>
  );
}

function Hint({ n, text }: { n: number; text: string }) {
  const t = useT();
  return (
    <View style={styles.hint} accessibilityLiveRegion="polite">
      <Icon name="bulb" size={16} color={colors.amber} />
      <View style={{ flex: 1, gap: 2 }}>
        <Eyebrow color={colors.amber}>{t('step.hintOf', { n, total: 2 })}</Eyebrow>
        <Text size={15}>{text}</Text>
      </View>
    </View>
  );
}

function Result({ q }: { q: QuestionView }) {
  const t = useT();
  const answer = q.type === 'mcq' && q.answer !== null ? q.options[q.answer] : q.solution;
  const right = q.answered && q.correct;
  const title = right ? t('step.correct') : q.answered ? t('step.wrong', { answer }) : t('step.revealed', { answer });
  return (
    <View style={[styles.result, { backgroundColor: right ? colors.mint : colors.peach }]} accessibilityLiveRegion="polite">
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
        <Icon name={right ? 'check' : 'close'} size={16} color={right ? colors.emeraldDeep : colors.danger} strokeWidth={2.25} />
        <Text size={16} weight="medium" color={right ? colors.emeraldDeep : colors.danger} style={{ flexShrink: 1 }}>
          {title}
        </Text>
      </View>
      {q.explanation ? <RichText size={14}>{q.explanation}</RichText> : null}
    </View>
  );
}

// Word order: tap words to build the sentence, tap again to take one back.
function WordOrder({ q, picked, setPicked, locked }: { q: QuestionView; picked: number[]; setPicked: (p: number[]) => void; locked: boolean }) {
  const t = useT();
  const shown = locked ? (q.picked ?? []) : picked.map((i) => q.tokens[i]);
  return (
    <View style={{ gap: 12 }}>
      <View style={[styles.sentence, locked && q.answered && { borderColor: q.correct ? colors.emerald : colors.danger }]}>
        {shown.length ? (
          shown.map((w, k) => (
            <Pressable key={`${w}-${k}`} disabled={locked} onPress={() => setPicked(picked.filter((_, j) => j !== k))} accessibilityRole="button" style={styles.word}>
              <Text size={15} weight="medium">
                {w}
              </Text>
            </Pressable>
          ))
        ) : (
          <Text size={14} color={colors.muted}>
            {t('step.tapWords')}
          </Text>
        )}
      </View>
      {!locked ? (
        <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
          {q.tokens.map((w, i) =>
            picked.includes(i) ? (
              <View key={i} style={[styles.word, styles.wordGhost]}>
                <Text size={15} color="transparent">
                  {w}
                </Text>
              </View>
            ) : (
              <Pressable key={i} onPress={() => setPicked([...picked, i])} accessibilityRole="button" style={[styles.word, styles.wordPick]}>
                <Text size={15} weight="medium">
                  {w}
                </Text>
              </Pressable>
            ),
          )}
          {picked.length ? <LinkButton label={t('step.clear')} icon="undo" onPress={() => setPicked([])} /> : null}
        </View>
      ) : null}
    </View>
  );
}

type Props = {
  q: QuestionView;
  eyebrow: string;
  busy: QuestionBusy;
  /** Read-only (a finished step shown again). */
  readOnly?: boolean;
  onAnswer: (input: AnswerInput) => void;
  onHint: () => void;
  onReveal: () => void;
};

// A practice question: pick, then Hint (max 2) or Answer; after both hints
// the learner may give up. The answer only arrives from the server once
// answered or revealed.
export function QuestionCard({ q, eyebrow, busy, readOnly, onAnswer, onHint, onReveal }: Props) {
  const t = useT();
  const [choice, setChoice] = useState<number | null>(null);
  const [picked, setPicked] = useState<number[]>([]);
  const locked = q.answered || q.revealed || !!readOnly;
  const ready = q.type === 'mcq' ? choice !== null : picked.length === q.tokens.length;

  const optionState = (i: number) => {
    if (!locked) return choice === i ? 'picked' : 'idle';
    if (q.answer === i) return 'right';
    if (q.choice === i && !q.correct) return 'wrong';
    return 'dim';
  };

  return (
    <View style={{ gap: 12 }}>
      <View style={styles.card}>
        <Eyebrow color={colors.accent}>{eyebrow}</Eyebrow>
        <Text size={18} weight="medium" tight style={{ lineHeight: 27 }}>
          {q.prompt}
        </Text>
        {q.type === 'mcq' ? (
          <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }} accessibilityRole="radiogroup">
            {q.options.map((o, i) => (
              <Option key={i} label={o} state={optionState(i)} disabled={locked || !!busy} onPress={() => setChoice(i)} />
            ))}
          </View>
        ) : (
          <WordOrder q={q} picked={picked} setPicked={setPicked} locked={locked} />
        )}
        {q.hints.map((h, i) => (
          <Hint key={i} n={i + 1} text={h} />
        ))}
        {q.answered || q.revealed ? <Result q={q} /> : null}
      </View>
      {!locked ? (
        <ActionRow left={q.hintsLeft === 0 ? <LinkButton label={t('step.giveUp')} icon="flag" onPress={onReveal} /> : undefined}>
          <PillButton
            label={q.hintsLeft === 0 ? t('step.hintsUsed') : q.hints.length ? t('step.hintsLeft', { n: q.hintsLeft }) : t('step.hint')}
            variant="secondary"
            disabled={q.hintsLeft === 0 || !!busy}
            loading={busy === 'hint'}
            onPress={onHint}
          />
          <PillButton
            label={t('step.answer')}
            disabled={!ready || !!busy}
            loading={busy === 'answer' || busy === 'reveal'}
            accessibilityHint={ready ? undefined : t('step.pickFirst')}
            onPress={() => onAnswer(q.type === 'mcq' ? { choice: choice! } : { tokens: picked.map((i) => q.tokens[i]) })}
          />
        </ActionRow>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.line, borderRadius: 24, padding: 20, gap: 16 },
  option: { flexDirection: 'row', alignItems: 'center', gap: 6, borderWidth: 1.5, borderRadius: radius.pill, paddingHorizontal: 18, paddingVertical: 9, minHeight: 44 },
  hint: { flexDirection: 'row', gap: 10, backgroundColor: colors.amberSoft, borderRadius: radius.xxl, padding: 14 },
  result: { borderRadius: radius.xxl, padding: 16, gap: 8 },
  sentence: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: 8,
    minHeight: 56,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderColor: colors.lineStrong,
    borderRadius: radius.xxl,
    padding: 8,
  },
  word: { borderRadius: radius.lg + 2, paddingHorizontal: 12, paddingVertical: 7, backgroundColor: colors.accentSoft },
  wordPick: { backgroundColor: colors.card, borderWidth: 1, borderColor: colors.lineStrong },
  wordGhost: { backgroundColor: colors.inkSoft },
});
