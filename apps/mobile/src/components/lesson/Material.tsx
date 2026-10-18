import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Block, ExampleRow, Material, Section } from '@/lib/api';
import { colors, fonts, radius } from '@/theme/tokens';
import { ResourceCard } from './ResourceCard';
import { RichText } from './RichText';

function Examples({ rows }: { rows: ExampleRow[] }) {
  return (
    <View style={styles.examples}>
      {rows.map((r, i) => (
        <View key={i} style={[styles.exampleRow, i > 0 && { borderTopWidth: 1, borderTopColor: colors.line }]}>
          <Text size={16} weight="medium" tight selectable>
            {r.en}
          </Text>
          <Text size={14} color={colors.muted} selectable>
            {r.id}
            {r.note ? ` — ${r.note}` : ''}
          </Text>
        </View>
      ))}
    </View>
  );
}

function Tip({ text }: { text: string }) {
  return (
    <View style={styles.tip}>
      <Icon name="sparkle" size={16} color={colors.accent} />
      <View style={{ flex: 1 }}>
        <RichText size={15}>{text}</RichText>
      </View>
    </View>
  );
}

// A practice question inside the material: tap an option, see why. Not graded.
export function InlineCheck({ block }: { block: Extract<Block, { type: 'check' }> }) {
  const t = useT();
  const [picked, setPicked] = useState<number>();
  const done = picked !== undefined;
  const right = picked === block.answer;
  return (
    <View style={styles.check}>
      <Eyebrow color={colors.accent}>{t('lesson.quickCheck')}</Eyebrow>
      <Text size={16} weight="medium" tight>
        {block.prompt}
      </Text>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }} accessibilityRole="radiogroup">
        {block.options.map((o, i) => {
          const isAnswer = done && i === block.answer;
          const isWrongPick = done && i === picked && !right;
          return (
            <Pressable
              key={i}
              onPress={() => setPicked(i)}
              accessibilityRole="radio"
              accessibilityState={{ selected: picked === i }}
              style={[
                styles.option,
                isAnswer && { backgroundColor: colors.mint, borderColor: colors.emerald },
                isWrongPick && { backgroundColor: colors.peach, borderColor: colors.danger },
              ]}
            >
              <Text size={15} weight="medium">
                {o}
              </Text>
            </Pressable>
          );
        })}
      </View>
      {done ? (
        <Text size={14} color={right ? colors.emeraldDeep : colors.danger} accessibilityLiveRegion="polite">
          {right ? t('lesson.correct') : t('lesson.notQuite')} <Text size={14} color={colors.ink}>{block.explain}</Text>
        </Text>
      ) : null}
    </View>
  );
}

function BlockView({ block }: { block: Block }) {
  switch (block.type) {
    case 'text':
      return <RichText>{block.text}</RichText>;
    case 'tip':
      return <Tip text={block.text} />;
    case 'examples':
      return <Examples rows={block.rows} />;
    case 'check':
      return <InlineCheck block={block} />;
  }
}

export function SectionView({ section, index }: { section: Section; index: number }) {
  return (
    <View style={{ gap: 16 }} nativeID={`section-${section.id}`}>
      <View style={{ gap: 6 }}>
        <Text size={13} color={colors.accent} style={{ fontFamily: fonts.mono }}>
          {String(index + 1).padStart(2, '0')}
        </Text>
        <Heading size={28}>{section.title}</Heading>
      </View>
      {section.blocks.map((b, i) => (
        <BlockView key={i} block={b} />
      ))}
    </View>
  );
}

// The lesson's reading material: lead, sections, real-world links and the
// starter flashcard deck.
export function MaterialView({
  material,
  onSaveCards,
  savingCards,
  cardsSaved,
}: {
  material: Material;
  onSaveCards?: () => void;
  savingCards?: boolean;
  cardsSaved?: boolean;
}) {
  const t = useT();
  return (
    <View style={{ gap: 44 }}>
      <Text size={19} color={colors.muted} style={{ lineHeight: 30 }}>
        {material.lead}
      </Text>
      {material.sections.map((s, i) => (
        <SectionView key={s.id} section={s} index={i} />
      ))}
      {material.resources.length > 0 ? (
        <View style={{ gap: 14 }}>
          <Heading size={24}>{t('lesson.resources')}</Heading>
          <View style={{ gap: 12 }}>
            {material.resources.map((r) => (
              <ResourceCard key={r.url} item={r} />
            ))}
          </View>
        </View>
      ) : null}
      {material.flashcards.length > 0 && onSaveCards ? (
        <View style={styles.cards}>
          <View style={{ flex: 1, gap: 4, minWidth: 200 }}>
            <Text weight="medium" size={17} tight>
              {t('lesson.starterCards')}
            </Text>
            <Text size={14} color={colors.muted}>
              {cardsSaved ? t('lesson.savedCards') : t('lesson.starterCardsText', { n: material.flashcards.length })}
            </Text>
          </View>
          <PillButton
            label={cardsSaved ? t('step.saved') : t('lesson.saveCards')}
            icon={cardsSaved ? 'check' : undefined}
            variant="secondary"
            size="sm"
            loading={savingCards}
            disabled={cardsSaved}
            onPress={onSaveCards}
          />
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  examples: { borderWidth: 1, borderColor: colors.line, borderRadius: radius.xxl + 4, backgroundColor: colors.card, overflow: 'hidden' },
  exampleRow: { paddingHorizontal: 16, paddingVertical: 12, gap: 2 },
  tip: { flexDirection: 'row', gap: 10, backgroundColor: colors.accentSoft, borderRadius: radius.xxl + 4, padding: 16 },
  check: { gap: 12, borderWidth: 1, borderColor: colors.line, borderStyle: 'dashed', borderRadius: radius.xxl + 4, padding: 16, backgroundColor: colors.card },
  option: { maxWidth: '100%', borderWidth: 1.5, borderColor: colors.line, borderRadius: radius.pill, paddingHorizontal: 16, paddingVertical: 8, backgroundColor: colors.card },
  cards: { flexDirection: 'row', flexWrap: 'wrap', alignItems: 'center', gap: 14, borderWidth: 1, borderColor: colors.line, borderRadius: 20, padding: 18, backgroundColor: colors.card },
});
