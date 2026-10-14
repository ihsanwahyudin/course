import { useCallback } from 'react';

import { en } from '@/i18n/en';
import { useT, type TKey } from '@/i18n';

import { ApiError } from './api';

// Translate an API error by its code; fall back to the server's message.
export function useErrorText() {
  const t = useT();
  return useCallback(
    (e: unknown, fallback: TKey = 'common.somethingWrong') => {
      if (e instanceof ApiError) {
        if (e.code in en.errors) return t(`errors.${e.code as keyof typeof en.errors}`);
        return e.message || t(fallback);
      }
      return t(fallback);
    },
    [t],
  );
}
