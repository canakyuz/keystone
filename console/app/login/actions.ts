'use server';

import { redirect } from 'next/navigation';
import { api } from '@/lib/api';
import { writeToken } from '@/lib/session';
import type { LoginResult } from '@/lib/types';

export interface SignInState {
  message: string | null;
  email: string;
}

export async function signIn(_: SignInState, form: FormData): Promise<SignInState> {
  const email = String(form.get('email') ?? '').trim();
  const password = String(form.get('password') ?? '');
  const tenantId = String(form.get('tenant_id') ?? '').trim();

  if (!email || !password) {
    return { message: 'Enter the email and password of your Keystone account.', email };
  }

  const result = await api<LoginResult>('/api/v1/auth/login', {
    method: 'POST',
    authenticated: false,
    body: { email, password, ...(tenantId ? { tenant_id: tenantId } : {}) },
  });

  if (!result.ok) {
    if (result.status === 401) return { message: 'That email and password do not match an account.', email };
    if (result.status === 403) {
      return { message: 'This account is suspended. An administrator of its tenant can reactivate it.', email };
    }
    return { message: result.message, email };
  }

  await writeToken(result.data.access_token, result.data.expires_in);
  redirect('/');
}
