import { useState } from 'react';
import { Platform, StyleSheet, TextInput, View, type NativeSyntheticEvent, type TextInputKeyPressEventData } from 'react-native';

import { Chip } from '@/components/ui/Bits';
import { useT } from '@/i18n';
import { IconButton } from '@/components/ui/PillButton';
import { Eyebrow } from '@/components/ui/Text';
import { colors, fonts, shadow } from '@/theme/tokens';

const MAX = 2000;

// Pill composer with a round accent send button. Enter sends on web
// (Shift+Enter for a new line).
export function Composer({
  value,
  onChange,
  onSend,
  disabled,
  suggestions = [],
}: {
  value: string;
  onChange: (v: string) => void;
  onSend: (text: string) => void;
  disabled?: boolean;
  suggestions?: string[];
}) {
  const t = useT();
  const [focused, setFocused] = useState(false);
  const trimmed = value.trim();
  const canSend = !disabled && trimmed.length > 0 && value.length <= MAX;
  const send = () => canSend && onSend(trimmed);
  const onKeyPress = (e: NativeSyntheticEvent<TextInputKeyPressEventData & { shiftKey?: boolean }>) => {
    if (Platform.OS === 'web' && e.nativeEvent.key === 'Enter' && !e.nativeEvent.shiftKey) {
      e.preventDefault();
      send();
    }
  };
  return (
    <View style={{ gap: 10 }}>
      {suggestions.length > 0 && !disabled ? (
        <View style={styles.suggestions}>
          {suggestions.map((s) => (
            <Chip key={s} label={s} onPress={() => onSend(s)} />
          ))}
        </View>
      ) : null}
      <View style={[styles.shell, { borderColor: focused ? colors.accent : colors.line, boxShadow: shadow.card }]}>
        <TextInput
          value={value}
          onChangeText={onChange}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onKeyPress={onKeyPress}
          placeholder={t('lesson.placeholder')}
          placeholderTextColor={colors.muted}
          multiline
          maxLength={MAX + 200}
          editable={!disabled}
          accessibilityLabel={t('lesson.messageLabel')}
          style={styles.input}
        />
        <View style={{ opacity: canSend ? 1 : 0.4 }}>
          <IconButton icon="send" label={t('lesson.send')} variant="accent" onPress={send} size={42} />
        </View>
      </View>
      {value.length > MAX - 200 ? (
        <Eyebrow color={value.length > MAX ? colors.danger : colors.muted} style={{ alignSelf: 'flex-end' }}>
          {value.length}/{MAX}
        </Eyebrow>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  suggestions: { flexDirection: 'row', flexWrap: 'wrap', gap: 6 },
  shell: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    gap: 8,
    backgroundColor: colors.card,
    borderWidth: 1,
    borderRadius: 26,
    paddingLeft: 18,
    paddingRight: 5,
    paddingVertical: 5,
  },
  input: {
    flex: 1,
    minHeight: 42,
    maxHeight: 120,
    paddingTop: 11,
    paddingBottom: 11,
    fontFamily: fonts.regular,
    fontSize: 15,
    lineHeight: 20,
    color: colors.ink,
    ...(Platform.OS === 'web' ? ({ outlineStyle: 'none' } as object) : null),
  },
});
