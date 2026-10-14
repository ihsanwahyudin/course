import { Linking, Platform } from 'react-native';

// Open an external link outside the app (a new tab on web, without giving
// the page access to ours).
export function openExternal(url: string) {
  if (!/^https:\/\//.test(url)) return;
  if (Platform.OS === 'web' && typeof window !== 'undefined') {
    window.open(url, '_blank', 'noopener,noreferrer');
    return;
  }
  void Linking.openURL(url);
}
