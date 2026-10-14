import type { ReactNode } from 'react';
import { View } from 'react-native';

import { SectionHeader } from '@/components/ui/Bits';
import { useLayout } from '@/components/ui/Screen';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { colors } from '@/theme/tokens';

// Page title. Desktop uses the site's SectionHeader (eyebrow pill + big
// heading); phones use a compact app header like the Rs apps (small eyebrow
// over a large title, actions on the right).
export function PageHeader({ label, title, description, aside, eyebrow }: { label: string; title: ReactNode; description?: string; aside?: ReactNode; eyebrow?: string }) {
  const { wide } = useLayout();
  if (wide) return <SectionHeader wide label={label} title={title} description={description} aside={aside} />;
  return (
    <View style={{ marginBottom: 20, gap: 6 }}>
      <View style={{ flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 }}>
        <View style={{ flex: 1, gap: 4 }}>
          <Eyebrow>{eyebrow ?? label}</Eyebrow>
          {typeof title === 'string' ? <Heading size={34}>{title}</Heading> : title}
        </View>
        {aside}
      </View>
      {description ? (
        <Text size={14} color={colors.muted} style={{ marginTop: 4 }}>
          {description}
        </Text>
      ) : null}
    </View>
  );
}
