import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Flashcard } from '@/lib/api';
import { colors } from '@/theme/tokens';

// One card at a time: tap to flip, arrows to move, then save the deck for
// spaced-repetition review.
export function FlashcardDeck({ cards, onSave, saving, saved }: { cards: Flashcard[]; onSave: () => void; saving?: boolean; saved?: boolean }) {
  const t = useT();
  const [i, setI] = useState(0);
  const [flipped, setFlipped] = useState(false);
  const card = cards[i];
  const go = (d: number) => {
    setI((i + d + cards.length) % cards.length);
    setFlipped(false);
  };
  return (
    <View style={styles.wrap}>
      <Pressable
        onPress={() => setFlipped((f) => !f)}
        accessibilityRole="button"
        accessibilityLabel={`${flipped ? card.back : card.front}. ${t('step.flip')}`}
        style={[styles.card, flipped && { backgroundColor: colors.accentSoft, borderColor: colors.accent }]}
      >
        <Eyebrow>{`${i + 1} / ${cards.length}`}</Eyebrow>
        <Text size={18} weight="medium" tight style={{ textAlign: 'center' }}>
          {flipped ? card.back : card.front}
        </Text>
        <Eyebrow>{t('step.flip')}</Eyebrow>
      </Pressable>
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
        <PillButton label="‹" size="sm" variant="secondary" onPress={() => go(-1)} />
        <PillButton label="›" size="sm" variant="secondary" onPress={() => go(1)} />
        <View style={{ flex: 1 }} />
        <PillButton
          label={saved ? t('step.saved') : t('step.saveAll')}
          icon={saved ? 'check' : undefined}
          size="sm"
          loading={saving}
          disabled={saved}
          onPress={onSave}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 10 },
  card: {
    minHeight: 150,
    borderWidth: 1.5,
    borderColor: colors.line,
    borderRadius: 20,
    padding: 18,
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 10,
    backgroundColor: colors.card,
  },
});
