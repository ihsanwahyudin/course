import { Redirect } from 'expo-router';
import { useState } from 'react';

import { AuthForm } from '@/components/AuthForm';
import { useAuth } from '@/lib/auth';

export default function LoginScreen() {
  // Only bounce users who arrive already signed in; after a fresh login the
  // form itself navigates to ?next=.
  const [alreadySignedIn] = useState(() => useAuth.getState().status === 'authenticated');
  if (alreadySignedIn) return <Redirect href="/dashboard" />;
  return <AuthForm mode="login" />;
}
