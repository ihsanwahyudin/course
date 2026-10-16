import { useState } from 'react';
import { Platform, Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import { colors, radius } from '@/theme/tokens';
import { LinkButton } from './StepParts';

// Checkpoint answers that are tapped, not typed: letter tiles to spell a
// word, tick-all options, and pairs to match. Drafts are strings so they
// keep in the store: picked indexes "2,0,1"; for match, the right item
// chosen for each left one, empty when none yet ("2,,0").

const pointer = Platform.OS === 'web' ? ({ cursor: 'pointer' } as object) : null;

export const indexesOf = (draft: string) => (draft ? draft.split(',').map(Number) : []);

/** Spell a word: tap letter tiles in order; tap a placed letter to take it back. */
export function LetterPicker({ tiles, picked, setPicked, disabled }: { tiles: string[]; picked: number[]; setPicked: (p: number[]) => void; disabled?: boolean }) {
  const t = useT();
  return (
    <View style={{ gap: 12 }}>
      <View style={styles.slot} accessibilityLabel={picked.map((i) => tiles[i]).join(' ')}>
        {picked.length ? (
          picked.map((i, k) => (
            <Pressable
              key={`${i}-${k}`}
              disabled={disabled}
              onPress={() => setPicked(picked.filter((_, j) => j !== k))}
              accessibilityRole="button"
              accessibilityLabel={t('checkpoint.removeLetter', { letter: tiles[i] })}
              style={[styles.letter, styles.letterPlaced, pointer]}
            >
              <Text size={20} weight="medium" color={colors.accent}>
                {tiles[i]}
              </Text>
            </Pressable>
          ))
        ) : (
          <Text size={14} color={colors.muted}>
            {t('checkpoint.tapLetters')}
          </Text>
        )}
      </View>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
        {tiles.map((l, i) =>
          picked.includes(i) ? (
            <View key={i} style={[styles.letter, styles.letterGhost]} />
          ) : (
            <Pressable key={i} disabled={disabled} onPress={() => setPicked([...picked, i])} accessibilityRole="button" accessibilityLabel={l} style={[styles.letter, styles.letterPick, pointer]}>
              <Text size={20} weight="medium">
                {l}
              </Text>
            </Pressable>
          ),
        )}
        {picked.length ? <LinkButton label={t('step.clear')} icon="undo" onPress={() => setPicked([])} /> : null}
      </View>
    </View>
  );
}

/** Tick every right option. */
export function MultiPicker({ options, picked, setPicked, disabled }: { options: string[]; picked: number[]; setPicked: (p: number[]) => void; disabled?: boolean }) {
  const toggle = (i: number) => setPicked(picked.includes(i) ? picked.filter((p) => p !== i) : [...picked, i].sort((a, b) => a - b));
  return (
    <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
      {options.map((o, i) => {
        const on = picked.includes(i);
        return (
          <Pressable
            key={i}
            disabled={disabled}
            onPress={() => toggle(i)}
            accessibilityRole="checkbox"
            accessibilityState={{ checked: on, disabled }}
            aria-checked={on}
            style={[styles.option, on ? styles.optionOn : null, pointer]}
          >
            <View style={[styles.box, on ? styles.boxOn : null]}>{on ? <Icon name="check" size={12} color={colors.white} strokeWidth={3} /> : null}</View>
            <Text size={16} weight="medium" color={on ? colors.accent : colors.ink}>
              {o}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

/** Match: pick a row on the left, then its pair on the right. */
export function MatchPicker({ left, right, draft, setDraft, disabled }: { left: string[]; right: string[]; draft: string; setDraft: (d: string) => void; disabled?: boolean }) {
  const t = useT();
  const picks = left.map((_, i) => {
    const v = draft.split(',')[i];
    return v === undefined || v === '' ? -1 : Number(v);
  });
  const [active, setActive] = useState(() => Math.max(0, picks.indexOf(-1)));
  const save = (next: number[]) => setDraft(next.map((p) => (p < 0 ? '' : String(p))).join(','));
  const choose = (r: number) => {
    // A right item already used elsewhere moves to the active row.
    const next = picks.map((p) => (p === r ? -1 : p));
    next[active] = r;
    save(next);
    const empty = next.findIndex((p, i) => p < 0 && i !== active);
    if (empty >= 0) setActive(empty);
  };
  const tapRow = (i: number) => {
    if (i === active && picks[i] >= 0) save(picks.map((p, j) => (j === i ? -1 : p)));
    setActive(i);
  };
  return (
    <View style={{ gap: 14 }}>
      <View style={{ gap: 8 }}>
        {left.map((l, i) => {
          const on = i === active;
          return (
            <Pressable
              key={i}
              disabled={disabled}
              onPress={() => tapRow(i)}
              accessibilityRole="button"
              accessibilityState={{ selected: on }}
              aria-pressed={on}
              accessibilityLabel={t('checkpoint.pairFor', { left: l })}
              style={[styles.row, on ? styles.rowOn : null, pointer]}
            >
              <Text size={16} weight="medium" style={{ flex: 1 }}>
                {l}
              </Text>
              <Icon name="arrowRight" size={16} color={colors.muted} />
              <View style={[styles.pairSlot, picks[i] >= 0 ? styles.pairSlotFull : null]}>
                <Text size={16} weight="medium" color={picks[i] >= 0 ? colors.accent : colors.muted}>
                  {picks[i] >= 0 ? right[picks[i]] : t('checkpoint.choosePair')}
                </Text>
              </View>
            </Pressable>
          );
        })}
      </View>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
        {right.map((r, i) => {
          const used = picks.includes(i);
          return (
            <Pressable key={i} disabled={disabled} onPress={() => choose(i)} accessibilityRole="button" style={[styles.option, used ? styles.optionUsed : null, pointer]}>
              <Text size={16} weight="medium" color={used ? colors.muted : colors.ink}>
                {r}
              </Text>
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  slot: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: 6,
    minHeight: 60,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderColor: colors.lineStrong,
    borderRadius: radius.xxl,
    padding: 8,
  },
  letter: { width: 44, height: 44, borderRadius: radius.xl, alignItems: 'center', justifyContent: 'center' },
  letterPick: { backgroundColor: colors.card, borderWidth: 1.5, borderColor: colors.lineStrong },
  letterPlaced: { backgroundColor: colors.accentSoft },
  letterGhost: { backgroundColor: colors.inkSoft },
  option: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    borderWidth: 1.5,
    borderColor: colors.lineStrong,
    backgroundColor: colors.card,
    borderRadius: radius.pill,
    paddingHorizontal: 16,
    paddingVertical: 9,
    minHeight: 44,
  },
  optionOn: { backgroundColor: colors.accentSoft, borderColor: colors.accent },
  optionUsed: { backgroundColor: colors.inkSoft, borderColor: colors.line },
  box: { width: 18, height: 18, borderRadius: 5, borderWidth: 1.5, borderColor: colors.lineStrong, alignItems: 'center', justifyContent: 'center' },
  boxOn: { backgroundColor: colors.accent, borderColor: colors.accent },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    borderWidth: 1.5,
    borderColor: colors.line,
    backgroundColor: colors.card,
    borderRadius: radius.xxl,
    paddingLeft: 16,
    paddingRight: 6,
    paddingVertical: 6,
    minHeight: 52,
  },
  rowOn: { borderColor: colors.accent },
  pairSlot: {
    flex: 1,
    minHeight: 38,
    justifyContent: 'center',
    borderRadius: radius.xl,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderColor: colors.lineStrong,
    paddingHorizontal: 12,
  },
  pairSlotFull: { borderStyle: 'solid', borderColor: colors.accentSoft, backgroundColor: colors.accentSoft },
});
