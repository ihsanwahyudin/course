import { Pressable, StyleSheet, View } from 'react-native';

import { useLocale, type Locale } from '@/i18n';
import { useAuth } from '@/lib/auth';
import { colors, fonts, radius } from '@/theme/tokens';
import { Text } from './Text';

// Compact ID | EN switch, like the one on the Rs site.
export function LanguageToggle({ compact }: { compact?: boolean }) {
  const locale = useLocale((s) => s.locale);
  const setLanguage = useAuth((s) => s.setLanguage);
  const options: Locale[] = ['id', 'en'];
  return (
    <View style={[styles.wrap, compact && { padding: 2 }]} accessibilityRole="radiogroup" accessibilityLabel="Language / Bahasa">
      {options.map((l) => {
        const on = l === locale;
        return (
          <Pressable
            key={l}
            onPress={() => !on && setLanguage(l)}
            accessibilityRole="radio"
            accessibilityState={{ checked: on }}
            accessibilityLabel={l === 'id' ? 'Bahasa Indonesia' : 'English'}
            style={[styles.item, compact && { paddingHorizontal: 8, paddingVertical: 4 }, on && styles.on]}
          >
            <Text size={12} color={on ? colors.ink : colors.muted} style={{ fontFamily: on ? fonts.semibold : fonts.medium }}>
              {l.toUpperCase()}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: 'row', alignItems: 'center', backgroundColor: colors.inkSoft, borderRadius: radius.pill, padding: 3 },
  item: { paddingHorizontal: 10, paddingVertical: 5, borderRadius: radius.pill },
  on: { backgroundColor: colors.card, boxShadow: '0px 1px 2px rgba(17,17,19,0.12)' },
});
