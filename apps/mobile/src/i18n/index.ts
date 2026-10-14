import { getLocales } from 'expo-localization';
import * as SecureStore from 'expo-secure-store';
import { useCallback } from 'react';
import { Platform } from 'react-native';
import { create } from 'zustand';

import { en, type Dict } from './en';
import { id } from './id';

export type Locale = 'en' | 'id';
export const LOCALES: Locale[] = ['en', 'id'];
const dicts: Record<Locale, Dict> = { en, id };

// "a.b.c" paths to every string in the dictionary, so t() keys are checked.
type Paths<T, P extends string = ''> = {
  [K in keyof T & string]: T[K] extends string ? `${P}${K}` : Paths<T[K], `${P}${K}.`>;
}[keyof T & string];
export type TKey = Paths<Dict>;
export type TVars = Record<string, string | number>;

export function translate(locale: Locale, key: TKey, vars?: TVars): string {
  const get = (d: Dict) => key.split('.').reduce<unknown>((o, k) => (o as Record<string, unknown> | undefined)?.[k], d) as string | undefined;
  let s = get(dicts[locale]) ?? get(en) ?? key;
  if (vars) for (const [k, v] of Object.entries(vars)) s = s.split(`{${k}}`).join(String(v));
  return s;
}

// First visit follows the device/browser language; "in" is the legacy code for Indonesian.
export function detectLocale(): Locale {
  try {
    const tag = (Platform.OS === 'web' && typeof navigator !== 'undefined' ? navigator.language : getLocales()[0]?.languageTag) ?? 'en';
    return /^(id|in)\b/i.test(tag) ? 'id' : 'en';
  } catch {
    return 'en';
  }
}

const KEY = 'fluentpath.locale';
const store = {
  get: async (): Promise<string | null> => {
    if (Platform.OS !== 'web') return SecureStore.getItemAsync(KEY);
    try {
      return window.localStorage.getItem(KEY);
    } catch {
      return null;
    }
  },
  set: async (v: Locale) => {
    if (Platform.OS !== 'web') return SecureStore.setItemAsync(KEY, v);
    try {
      window.localStorage.setItem(KEY, v);
    } catch {
      // private mode etc. — the choice just won't persist
    }
  },
};

type LocaleState = {
  locale: Locale;
  // Restore a saved choice from this device, if any.
  restore: () => Promise<void>;
  setLocale: (l: Locale) => void;
};

export const useLocale = create<LocaleState>((set) => ({
  locale: detectLocale(),
  restore: async () => {
    const saved = await store.get().catch(() => null);
    if (saved === 'en' || saved === 'id') set({ locale: saved });
  },
  setLocale: (locale) => {
    set({ locale });
    void store.set(locale);
    if (Platform.OS === 'web' && typeof document !== 'undefined') document.documentElement.lang = locale;
  },
}));

export const currentLocale = () => useLocale.getState().locale;

export function useT() {
  const locale = useLocale((s) => s.locale);
  return useCallback((key: TKey, vars?: TVars) => translate(locale, key, vars), [locale]);
}

// BCP-47 tag for dates and numbers.
export const dateLocale = (l: Locale) => (l === 'id' ? 'id-ID' : 'en-GB');

export function useFormatDate() {
  const locale = useLocale((s) => s.locale);
  return useCallback(
    (iso: string | Date, opts: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'short', year: 'numeric' }) => new Date(iso).toLocaleDateString(dateLocale(locale), opts),
    [locale],
  );
}
