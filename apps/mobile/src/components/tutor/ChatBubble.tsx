import { useEffect, useState } from 'react';
import { Animated, Easing, Platform, StyleSheet, View } from 'react-native';

import { GradientRing } from '@/components/ui/Decor';
import { Eyebrow, Text } from '@/components/ui/Text';
import { phaseLabel, type Message, type Phase } from '@/lib/api';
import { colors, radius, shadow } from '@/theme/tokens';

const kindBg: Record<string, string> = {
  assessment_intro: colors.lavender,
  review_intro: colors.peach,
  result: colors.mint,
};

export function ChatBubble({ message, showAvatar, pending }: { message: Pick<Message, 'role' | 'content' | 'metadata'>; showAvatar: boolean; pending?: boolean }) {
  const mine = message.role === 'user';
  const special = message.metadata?.kind ? kindBg[message.metadata.kind] : undefined;
  return (
    <View style={[styles.row, mine ? styles.rowMine : null]}>
      {!mine ? <View style={styles.avatar}>{showAvatar ? <GradientRing size={30} /> : null}</View> : null}
      <View
        style={[
          styles.bubble,
          mine
            ? { backgroundColor: colors.ink, borderTopRightRadius: 6 }
            : { backgroundColor: special ?? colors.card, borderTopLeftRadius: 6, borderWidth: 1, borderColor: colors.line, boxShadow: shadow.card },
          pending && { opacity: 0.6 },
        ]}
      >
        <Text size={15} color={mine ? colors.white : colors.ink} style={{ lineHeight: 23 }} selectable>
          {message.content}
        </Text>
        {!mine && message.metadata?.action === 'ask_retry' ? (
          <Eyebrow color={colors.accent} style={{ marginTop: 8 }}>
            Try again ↺
          </Eyebrow>
        ) : null}
      </View>
    </View>
  );
}

export function PhaseDivider({ phase }: { phase: Phase }) {
  return (
    <View style={styles.divider} accessibilityRole="header">
      <View style={styles.dividerLine} />
      <View style={styles.dividerPill}>
        <View style={{ width: 6, height: 6, borderRadius: 3, backgroundColor: colors.accent }} />
        <Eyebrow>{phase === 'GUIDED_PRACTICE' ? 'Guided practice' : phase === 'INDEPENDENT_PRACTICE' ? 'Independent practice' : phaseLabel[phase]}</Eyebrow>
      </View>
      <View style={styles.dividerLine} />
    </View>
  );
}

function Dot({ delay }: { delay: number }) {
  const v = useState(() => new Animated.Value(0))[0];
  useEffect(() => {
    const a = Animated.loop(
      Animated.sequence([
        Animated.timing(v, { toValue: 1, duration: 400, delay, easing: Easing.out(Easing.quad), useNativeDriver: Platform.OS !== 'web' }),
        Animated.timing(v, { toValue: 0, duration: 400, easing: Easing.in(Easing.quad), useNativeDriver: Platform.OS !== 'web' }),
        Animated.delay(400 - delay),
      ]),
    );
    a.start();
    return () => a.stop();
  }, [v, delay]);
  return <Animated.View style={[styles.typingDot, { opacity: v.interpolate({ inputRange: [0, 1], outputRange: [0.3, 1] }), transform: [{ translateY: v.interpolate({ inputRange: [0, 1], outputRange: [0, -3] }) }] }]} />;
}

export function TypingBubble() {
  return (
    <View style={styles.row} accessibilityLabel="Tutor is typing" accessibilityLiveRegion="polite">
      <View style={styles.avatar}>
        <GradientRing size={30} />
      </View>
      <View style={[styles.bubble, { backgroundColor: colors.card, borderTopLeftRadius: 6, borderWidth: 1, borderColor: colors.line, flexDirection: 'row', gap: 5, paddingVertical: 16 }]}>
        <Dot delay={0} />
        <Dot delay={130} />
        <Dot delay={260} />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'flex-start', gap: 10, maxWidth: '100%' },
  rowMine: { justifyContent: 'flex-end' },
  avatar: { width: 30 },
  bubble: { maxWidth: '82%', borderRadius: 20, paddingHorizontal: 16, paddingVertical: 12 },
  divider: { flexDirection: 'row', alignItems: 'center', gap: 10, marginVertical: 8 },
  dividerLine: { flex: 1, height: 1, backgroundColor: colors.line },
  dividerPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    borderWidth: 1,
    borderColor: colors.line,
    backgroundColor: colors.card,
    borderRadius: radius.pill,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  typingDot: { width: 7, height: 7, borderRadius: 4, backgroundColor: colors.accent },
});
