import { type ReactNode } from 'react';
import { RefreshControl, ScrollView, StyleSheet, useWindowDimensions, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { colors, layout } from '@/theme/tokens';
import { BOTTOM_TAB_HEIGHT, useChrome } from './chrome';
import { DotPattern } from './Decor';

export function useLayout() {
  const { width } = useWindowDimensions();
  const wide = width >= layout.wideBreakpoint;
  const gutter = wide ? layout.gutterWide : layout.gutter;
  const gap = wide ? layout.gapWide : layout.gap;
  const contentWidth = Math.min(width, layout.maxWidth) - gutter * 2;
  return { width, wide, gutter, gap, contentWidth, cols: wide ? 4 : 2 };
}

// Width of a tile spanning `span` of `cols` columns in a bento grid.
export function span(contentWidth: number, gap: number, cols: number, n: number) {
  const unit = (contentWidth - gap * (cols - 1)) / cols;
  return unit * n + gap * (n - 1);
}

// Dotted canvas + centred max-w-6xl column, cleared for the floating nav.
export function Screen({ children, onRefresh, refreshing = false, maxWidth = layout.maxWidth }: { children: ReactNode; onRefresh?: () => void; refreshing?: boolean; maxWidth?: number }) {
  const insets = useSafeAreaInsets();
  const { wide, gutter } = useLayout();
  const chrome = useChrome();
  return (
    <View style={styles.root}>
      <DotPattern />
      <ScrollView
        contentContainerStyle={{
          // Clear whichever chrome is on screen: the floating top nav, or the
          // bottom tab bar (phones, logged in) where the page has its own header.
          paddingTop: insets.top + (chrome.topNav ? (wide ? layout.navClearanceWide : layout.navClearance) : 20),
          paddingBottom: chrome.bottomTabs ? Math.max(insets.bottom, 8) + BOTTOM_TAB_HEIGHT + 28 : insets.bottom + 48,
          paddingHorizontal: gutter,
          alignItems: 'center',
        }}
        refreshControl={onRefresh ? <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.accent} /> : undefined}
      >
        <View style={{ width: '100%', maxWidth: maxWidth - gutter * 2 }}>{children}</View>
      </ScrollView>
    </View>
  );
}

export const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.canvas },
});
