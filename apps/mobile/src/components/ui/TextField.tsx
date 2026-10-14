import { forwardRef, useState } from 'react';
import { Platform, StyleSheet, TextInput, View, type TextInputProps } from 'react-native';

import { colors, fonts, radius } from '@/theme/tokens';
import { Eyebrow, Text } from './Text';

type Props = TextInputProps & { label: string; error?: string; hint?: string };

export const TextField = forwardRef<TextInput, Props>(function TextField({ label, error, hint, style, ...rest }, ref) {
  const [focused, setFocused] = useState(false);
  return (
    <View style={{ gap: 6 }}>
      <Eyebrow>{label}</Eyebrow>
      <TextInput
        ref={ref}
        placeholderTextColor={colors.muted}
        accessibilityLabel={label}
        {...rest}
        onFocus={(e) => {
          setFocused(true);
          rest.onFocus?.(e);
        }}
        onBlur={(e) => {
          setFocused(false);
          rest.onBlur?.(e);
        }}
        style={[styles.input, { borderColor: error ? colors.danger : focused ? colors.accent : colors.line }, style]}
      />
      {error ? (
        <Text size={12} color={colors.danger}>
          {error}
        </Text>
      ) : hint ? (
        <Text size={12} color={colors.muted}>
          {hint}
        </Text>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  input: {
    backgroundColor: colors.canvas,
    borderWidth: 1,
    borderRadius: radius.xxl,
    paddingHorizontal: 14,
    paddingVertical: 12,
    minHeight: 48,
    fontFamily: fonts.regular,
    fontSize: 15,
    color: colors.ink,
    ...(Platform.OS === 'web' ? ({ outlineStyle: 'none' } as object) : null),
  },
});
