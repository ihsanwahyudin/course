import { useRouter } from 'expo-router';
import { useState } from 'react';
import { StyleSheet, View } from 'react-native';

import { MemorySection } from '@/components/account/MemorySection';
import { PageHeader } from '@/components/PageHeader';
import { Card } from '@/components/ui/Card';
import { Avatar } from '@/components/ui/FloatingNav';
import { Icon } from '@/components/ui/Icon';
import { PillButton } from '@/components/ui/PillButton';
import { Screen, useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { LanguageToggle } from '@/components/ui/LanguageToggle';
import { useFormatDate, useT } from '@/i18n';
import { useAuth } from '@/lib/auth';
import { useDashboard } from '@/lib/queries';
import { colors, fonts } from '@/theme/tokens';

export default function AccountScreen() {
  const router = useRouter();
  const user = useAuth((s) => s.user);
  const logout = useAuth((s) => s.logout);
  const { data } = useDashboard();
  const { wide } = useLayout();
  const [busy, setBusy] = useState(false);
  const t = useT();
  const formatDate = useFormatDate();
  if (!user) return null;

  const rows: [string, string][] = [
    [t('account.email'), user.email],
    [t('account.memberSince'), formatDate(user.createdAt, { day: 'numeric', month: 'long', year: 'numeric' })],
    [t('account.courses'), String(data?.courses.length ?? '–')],
    [t('account.skillsMastered'), String(data?.totalMastered ?? '–')],
  ];

  return (
    <Screen maxWidth={760}>
      <PageHeader label={t('account.title')} title={wide ? t('account.titleWide') : t('account.title')} />
      <View style={{ gap: wide ? 16 : 12 }}>
        <Card reveal={0} padding={wide ? 32 : 24} style={{ gap: 24 }}>
          <View style={styles.row}>
            <Avatar name={user.displayName} size={64} />
            <View style={{ flex: 1 }}>
              <Heading size={30}>{user.displayName}</Heading>
              <Text color={colors.muted}>{user.email}</Text>
            </View>
          </View>
          <View style={{ borderTopWidth: 1, borderTopColor: colors.line }}>
            {rows.map(([k, v]) => (
              <View key={k} style={styles.item}>
                <Text color={colors.muted}>{k}</Text>
                <Text style={{ fontFamily: fonts.mono, fontSize: 13 }}>{v}</Text>
              </View>
            ))}
          </View>
        </Card>
        <MemorySection reveal={1} />
        <Card reveal={2} style={{ gap: 14 }}>
          <View style={[styles.row, { justifyContent: 'space-between' }]}>
            <Eyebrow>{t('common.language')}</Eyebrow>
            <LanguageToggle />
          </View>
          <Text color={colors.muted}>{t('account.languageText')}</Text>
        </Card>
        <Card reveal={3} style={{ gap: 14 }}>
          <View style={styles.row}>
            <Icon name="logout" size={18} color={colors.muted} />
            <Eyebrow>{t('account.session')}</Eyebrow>
          </View>
          <Text color={colors.muted}>{t('account.logoutText')}</Text>
          <PillButton
            label={t('account.logout')}
            variant="secondary"
            loading={busy}
            onPress={async () => {
              setBusy(true);
              await logout();
              router.replace('/');
            }}
          />
        </Card>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 14 },
  item: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 14, borderBottomWidth: 1, borderBottomColor: colors.line },
});
