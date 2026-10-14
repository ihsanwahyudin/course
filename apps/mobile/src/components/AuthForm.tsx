import { Link, useLocalSearchParams, useRouter, type Href } from 'expo-router';
import { useRef, useState } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type TextInput } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { Card } from '@/components/ui/Card';
import { DotPattern, GradientRing, MeshBlob } from '@/components/ui/Decor';
import { GradientText } from '@/components/ui/GradientText';
import { PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { TextField } from '@/components/ui/TextField';
import { ApiError } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { colors, fonts } from '@/theme/tokens';

// Only allow in-app return paths (no open redirects via ?next=).
function safeNext(next: unknown): Href {
  return typeof next === 'string' && next.startsWith('/') && !next.startsWith('//') && !next.startsWith('/login') && !next.startsWith('/register')
    ? (next as Href)
    : '/dashboard';
}

export function AuthForm({ mode }: { mode: 'login' | 'register' }) {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { next } = useLocalSearchParams<{ next?: string }>();
  const login = useAuth((s) => s.login);
  const register = useAuth((s) => s.register);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<{ field?: 'name' | 'email' | 'password'; message: string }>();
  const emailRef = useRef<TextInput>(null);
  const passwordRef = useRef<TextInput>(null);
  const isRegister = mode === 'register';

  const submit = async () => {
    setError(undefined);
    if (isRegister && !name.trim()) return setError({ field: 'name', message: 'Tell us what to call you.' });
    if (!email.trim()) return setError({ field: 'email', message: 'Enter your email.' });
    if (password.length < (isRegister ? 8 : 1)) return setError({ field: 'password', message: isRegister ? 'Use at least 8 characters.' : 'Enter your password.' });
    setBusy(true);
    try {
      if (isRegister) await register(name.trim(), email.trim(), password);
      else await login(email.trim(), password);
      router.replace(safeNext(next));
    } catch (e) {
      const err = e instanceof ApiError ? e : undefined;
      const field = err?.code === 'invalid_email' || err?.code === 'email_taken' ? 'email' : err?.code === 'invalid_password' ? 'password' : err?.code === 'invalid_name' ? 'name' : undefined;
      setError({ field, message: err?.message ?? 'Something went wrong. Please try again.' });
    } finally {
      setBusy(false);
    }
  };

  const other = isRegister ? { label: 'Log in', href: '/login', prompt: 'Already have an account?' } : { label: 'Create one', href: '/register', prompt: 'New to FluentPath?' };

  return (
    <View style={styles.root}>
      <DotPattern />
      <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <ScrollView contentContainerStyle={[styles.scroll, { paddingTop: insets.top + 112, paddingBottom: insets.bottom + 40 }]} keyboardShouldPersistTaps="handled">
          <Card reveal={0} padding={28} spotlight={false} style={styles.card} decoration={<MeshBlob color={colors.lavender} size={280} style={{ right: -120, top: -140 }} />}>
            <View style={{ gap: 18 }}>
              <GradientRing size={48} />
              <View style={{ gap: 8 }}>
                <Eyebrow>{isRegister ? 'Create account' : 'Welcome back'}</Eyebrow>
                <Heading size={34}>
                  {isRegister ? 'Start your ' : 'Continue your '}
                  {Platform.OS === 'web' ? (
                    <GradientText style={{ fontFamily: fonts.semibold, fontSize: 34, lineHeight: 35, letterSpacing: -1.5 }}>roadmap.</GradientText>
                  ) : (
                    'roadmap.'
                  )}
                </Heading>
                <Text color={colors.muted}>
                  {isRegister ? 'Free account. Your progress is saved to your dashboard across devices.' : 'Log in to pick up your lessons where you left off.'}
                </Text>
              </View>
            </View>
            <View style={{ gap: 14 }}>
              {isRegister ? (
                <TextField
                  label="Name"
                  value={name}
                  onChangeText={setName}
                  placeholder="Rina"
                  autoComplete="name"
                  textContentType="name"
                  returnKeyType="next"
                  onSubmitEditing={() => emailRef.current?.focus()}
                  error={error?.field === 'name' ? error.message : undefined}
                />
              ) : null}
              <TextField
                ref={emailRef}
                label="Email"
                value={email}
                onChangeText={setEmail}
                placeholder="you@example.com"
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="email-address"
                autoComplete="email"
                textContentType={isRegister ? 'emailAddress' : 'username'}
                returnKeyType="next"
                onSubmitEditing={() => passwordRef.current?.focus()}
                error={error?.field === 'email' ? error.message : undefined}
              />
              <TextField
                ref={passwordRef}
                label="Password"
                value={password}
                onChangeText={setPassword}
                placeholder={isRegister ? 'At least 8 characters' : 'Your password'}
                secureTextEntry
                autoComplete={isRegister ? 'new-password' : 'current-password'}
                textContentType={isRegister ? 'newPassword' : 'password'}
                returnKeyType="go"
                onSubmitEditing={submit}
                error={error?.field === 'password' ? error.message : undefined}
              />
              {error && !error.field ? (
                <View style={styles.error}>
                  <Text size={14}>{error.message}</Text>
                </View>
              ) : null}
            </View>
            <PillButton label={isRegister ? 'Create account' : 'Log in'} icon="arrowUpRight" onPress={submit} loading={busy} style={{ alignSelf: 'stretch' }} />
            <Text size={14} color={colors.muted} style={{ textAlign: 'center' }}>
              {other.prompt}{' '}
              <Link href={{ pathname: other.href as '/login', params: next ? { next } : {} }} replace style={{ color: colors.accent, fontFamily: fonts.medium }}>
                {other.label}
              </Link>
            </Text>
          </Card>
        </ScrollView>
      </KeyboardAvoidingView>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.canvas },
  scroll: { paddingHorizontal: 16, alignItems: 'center' },
  card: { width: '100%', maxWidth: 440, gap: 24 },
  error: { backgroundColor: colors.peach, borderRadius: 16, padding: 14 },
});
