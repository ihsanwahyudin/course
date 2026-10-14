import { createElement, useState } from 'react';
import { Image, Platform, Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/ui/Icon';
import { Eyebrow, Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { CuratedResource } from '@/lib/api';
import { openExternal } from '@/lib/links';
import { colors, radius } from '@/theme/tokens';

type Item = CuratedResource & { videoId?: string };

const youTubeId = (url: string) => url.match(/(?:v=|youtu\.be\/|\/shorts\/|\/embed\/)([A-Za-z0-9_-]{11})/)?.[1];

// A real-world link: YouTube videos play inline on web (privacy-enhanced
// embed) and open YouTube on phones.
export function ResourceCard({ item, compact }: { item: Item; compact?: boolean }) {
  const t = useT();
  const [playing, setPlaying] = useState(false);
  const [hovered, setHovered] = useState(false);
  const videoId = item.kind === 'video' ? (item.videoId ?? youTubeId(item.url)) : undefined;

  const media = videoId ? (
    playing && Platform.OS === 'web' ? (
      <View style={styles.media}>
        {createElement('iframe', {
          src: `https://www.youtube-nocookie.com/embed/${videoId}?autoplay=1&rel=0`,
          title: item.title,
          allow: 'autoplay; encrypted-media; picture-in-picture',
          allowFullScreen: true,
          style: { border: 0, width: '100%', height: '100%' },
        })}
      </View>
    ) : (
      <Pressable
        onPress={() => (Platform.OS === 'web' ? setPlaying(true) : openExternal(item.url))}
        accessibilityRole="button"
        accessibilityLabel={`${t('lesson.watch')}: ${item.title}`}
        style={styles.media}
      >
        <Image source={{ uri: `https://i.ytimg.com/vi/${videoId}/hqdefault.jpg` }} style={StyleSheet.absoluteFill} resizeMode="cover" accessibilityIgnoresInvertColors />
        <View style={styles.play}>
          <Icon name="play" size={20} color={colors.white} />
        </View>
      </Pressable>
    )
  ) : null;

  return (
    <View style={[styles.card, compact && { borderRadius: radius.xxl }]}>
      {media}
      <Pressable
        onPress={() => openExternal(item.url)}
        onHoverIn={() => setHovered(true)}
        onHoverOut={() => setHovered(false)}
        accessibilityRole="link"
        style={[styles.body, hovered && { backgroundColor: colors.canvas }]}
      >
        <View style={{ flex: 1, gap: 2 }}>
          <Eyebrow>
            {t(item.kind === 'video' ? 'lesson.video' : item.kind === 'image' ? 'lesson.image' : 'lesson.article')} · {item.source}
          </Eyebrow>
          <Text weight="medium" size={compact ? 14 : 15} tight numberOfLines={2}>
            {item.title}
          </Text>
          {item.note ? (
            <Text size={13} color={colors.muted} numberOfLines={2}>
              {item.note}
            </Text>
          ) : null}
        </View>
        <Icon name="arrowUpRight" size={16} color={colors.muted} />
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { borderWidth: 1, borderColor: colors.line, borderRadius: 20, overflow: 'hidden', backgroundColor: colors.card },
  media: { width: '100%', aspectRatio: 16 / 9, backgroundColor: colors.inkSoft, alignItems: 'center', justifyContent: 'center' },
  play: { width: 52, height: 52, borderRadius: 26, backgroundColor: 'rgba(17,17,19,0.75)', alignItems: 'center', justifyContent: 'center' },
  body: { flexDirection: 'row', alignItems: 'center', gap: 10, padding: 14 },
});
