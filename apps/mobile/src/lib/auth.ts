import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';
import { create } from 'zustand';

import { api, setAuthToken, setUnauthorizedHandler, type User } from './api';
import { queryClient } from './queries';

// Native keeps the bearer token in the OS keychain/keystore. Web never sees a
// token: the server sets an HttpOnly cookie instead.
const TOKEN_KEY = 'fluentpath.session';

const tokenStore = {
  get: () => (Platform.OS === 'web' ? Promise.resolve(null) : SecureStore.getItemAsync(TOKEN_KEY)),
  set: (t: string) => (Platform.OS === 'web' ? Promise.resolve() : SecureStore.setItemAsync(TOKEN_KEY, t)),
  clear: () => (Platform.OS === 'web' ? Promise.resolve() : SecureStore.deleteItemAsync(TOKEN_KEY)),
};

type AuthState = {
  status: 'loading' | 'anonymous' | 'authenticated';
  user: User | null;
  init: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (displayName: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

async function signedIn(user: User, token?: string) {
  if (token) {
    setAuthToken(token);
    await tokenStore.set(token);
  }
  // Anything cached while anonymous (or as someone else) is stale now.
  queryClient.clear();
  useAuth.setState({ status: 'authenticated', user });
}

async function signedOut() {
  setAuthToken(null);
  await tokenStore.clear().catch(() => {});
  queryClient.clear();
  useAuth.setState({ status: 'anonymous', user: null });
}

export const useAuth = create<AuthState>(() => ({
  status: 'loading',
  user: null,

  init: async () => {
    const token = await tokenStore.get().catch(() => null);
    setAuthToken(token);
    try {
      const { user } = await api.me();
      useAuth.setState({ status: 'authenticated', user });
    } catch {
      await signedOut();
    }
  },

  login: async (email, password) => {
    const { user, token } = await api.login({ email, password });
    await signedIn(user, token);
  },

  register: async (displayName, email, password) => {
    const { user, token } = await api.register({ displayName, email, password });
    await signedIn(user, token);
  },

  logout: async () => {
    await api.logout().catch(() => {});
    await signedOut();
  },
}));

// A 401 on any authenticated call means the session expired or was revoked.
setUnauthorizedHandler(() => {
  if (useAuth.getState().status === 'authenticated') void signedOut();
});
