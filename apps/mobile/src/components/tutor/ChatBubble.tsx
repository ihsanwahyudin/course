import { useEffect, useState } from 'react';
import { Animated, Easing, Platform, StyleSheet, View } from 'react-native';

import { GradientRing } from '@/components/ui/Decor';
import { Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { Message } from '@/lib/api';
import { colors, shadow } from '@/theme/tokens';

const kindBg: Record<string, string> = {
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
      </View>
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
  const t = useT();
  return (
    <View style={styles.row} accessibilityLabel={t('lesson.typing')} accessibilityLiveRegion="polite">
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
  bubble: { maxWidth: '88%', borderRadius: 20, paddingHorizontal: 16, paddingVertical: 12 },
  typingDot: { width: 7, height: 7, borderRadius: 4, backgroundColor: colors.accent },
});
