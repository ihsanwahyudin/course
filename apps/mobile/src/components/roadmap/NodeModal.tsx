import { useRouter } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import { AccessibilityInfo, Animated, Easing, Modal, Platform, Pressable, ScrollView, StyleSheet, useWindowDimensions, View } from 'react-native';

import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Icon } from '@/components/ui/Icon';
import { useT } from '@/i18n';
import { useErrorText } from '@/lib/errors';
import { useNode, useRoadmap } from '@/lib/queries';
import { colors, layout, radius, shadow } from '@/theme/tokens';
import { NodeDetailView } from './NodeDetail';

const expoOut = Easing.bezier(0.16, 1, 0.3, 1);

// A lesson's details over the roadmap, in a dialog like ihsan.my.id's
// project dialog: the backdrop blurs in and the panel rises; closing plays it
// back. The content is the lesson page's.
export function NodeModal({ course, nodeId, onSwitch, onClose }: { course: string; nodeId: string; onSwitch: (id: string) => void; onClose: () => void }) {
  const t = useT();
  const router = useRouter();
  const errorText = useErrorText();
  const { width, height } = useWindowDimensions();
  const wide = width >= layout.wideBreakpoint;
  const gap = wide ? layout.gapWide : layout.gap;
  const outer = wide ? 24 : 10;
  const inner = wide ? 24 : 12;
  const panelW = Math.min(width - outer * 2, layout.maxWidth);
  const { data: node, isPending, isError, error, refetch } = useNode(course, nodeId);
  const goal = useRoadmap(course).data?.profile.goal;

  const show = useState(() => new Animated.Value(0))[0];
  const scrollRef = useRef<ScrollView>(null);
  const closing = useRef(false);
  const [reduce, setReduce] = useState(false);

  useEffect(() => {
    let live = true;
    AccessibilityInfo.isReduceMotionEnabled()
      .catch(() => false)
      .then((r) => {
        if (!live) return;
        setReduce(r);
        Animated.timing(show, { toValue: 1, duration: r ? 200 : 700, easing: expoOut, useNativeDriver: false }).start();
      });
    return () => {
      live = false;
    };
  }, [show]);

  const close = useCallback(() => {
    if (closing.current) return;
    closing.current = true;
    Animated.timing(show, { toValue: 0, duration: reduce ? 150 : 350, easing: Easing.in(Easing.quad), useNativeDriver: false }).start(onClose);
  }, [show, reduce, onClose]);

  // Freeze the page behind the dialog, and close it with Escape (web).
  useEffect(() => {
    if (Platform.OS !== 'web') return;
    const root = document.documentElement;
    const overflow = root.style.overflow;
    root.style.overflow = 'hidden';
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
    };
    window.addEventListener('keydown', onKey);
    return () => {
      root.style.overflow = overflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [close]);

  // Another lesson in the same dialog starts at the top.
  useEffect(() => {
    scrollRef.current?.scrollTo({ y: 0, animated: false });
  }, [nodeId]);

  const leave = (path: string) => {
    onClose();
    router.push(path as never);
  };

  const rise = reduce
    ? {}
    : { transform: [{ translateY: show.interpolate({ inputRange: [0, 1], outputRange: [40, 0] }) }, { scale: show.interpolate({ inputRange: [0, 1], outputRange: [0.97, 1] }) }] };

  return (
    <Modal visible transparent animationType="none" onRequestClose={close} statusBarTranslucent>
      <Animated.View style={[StyleSheet.absoluteFill, styles.backdrop, { opacity: show }]} />
      <ScrollView ref={scrollRef} style={StyleSheet.absoluteFill} contentContainerStyle={[styles.scroll, { minHeight: height, padding: outer }]}>
        <Pressable style={StyleSheet.absoluteFill} onPress={close} accessibilityLabel={t('common.close')} />
        <Animated.View accessibilityViewIsModal style={[styles.panel, { width: panelW, padding: inner, opacity: show }, rise]}>
          <View style={styles.head}>
            <CloseButton onPress={close} label={t('common.close')} />
          </View>
          {isPending ? (
            <LoadingState label={t('node.loading')} />
          ) : isError ? (
            <ErrorState title={t('node.loadError')} message={errorText(error)} onRetry={refetch} />
          ) : (
            <View key={nodeId}>
              <NodeDetailView
                node={node}
                width={panelW - inner * 2}
                wide={wide}
                gap={gap}
                goal={goal?.label}
                onCourse={close}
                onNode={(c, id) => (c === course ? onSwitch(id) : leave(`/courses/${c}/${id}`))}
                onGo={leave}
              />
            </View>
          )}
        </Animated.View>
      </ScrollView>
    </Modal>
  );
}

// Round close button; it turns a quarter on hover, like the site's. Focus
// starts here when the dialog opens.
function CloseButton({ onPress, label }: { onPress: () => void; label: string }) {
  const [hovered, setHovered] = useState(false);
  const ref = useRef<View>(null);
  useEffect(() => {
    (ref.current as unknown as { focus?: (o?: object) => void } | null)?.focus?.({ preventScroll: true });
  }, []);
  return (
    <Pressable
      ref={ref}
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="button"
      accessibilityLabel={label}
      style={[
        styles.close,
        { transform: [{ rotate: hovered ? '90deg' : '0deg' }] },
        Platform.OS === 'web' && ({ transitionProperty: 'transform', transitionDuration: '500ms', transitionTimingFunction: 'cubic-bezier(0.16,1,0.3,1)' } as object),
      ]}
    >
      <Icon name="close" size={16} color={colors.ink} strokeWidth={2} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  backdrop: { backgroundColor: 'rgba(17,17,19,0.35)', ...(Platform.OS === 'web' ? ({ backdropFilter: 'blur(12px)' } as object) : {}) },
  scroll: { alignItems: 'center', justifyContent: 'flex-start' },
  panel: { backgroundColor: colors.canvas, borderRadius: radius.card + 4, borderWidth: 1, borderColor: colors.line, boxShadow: shadow.cardHover, gap: 12 },
  head: { flexDirection: 'row', justifyContent: 'flex-end' },
  close: { width: 40, height: 40, borderRadius: 20, backgroundColor: colors.card, alignItems: 'center', justifyContent: 'center', boxShadow: shadow.card },
});
