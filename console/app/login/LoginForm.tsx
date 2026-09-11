'use client';

import { useActionState } from 'react';
import { signIn, type SignInState } from './actions';

const initial: SignInState = { message: null, email: '' };

export function LoginForm({ notice }: { notice: string | null }) {
  const [state, action, pending] = useActionState(signIn, initial);

  return (
    <form action={action} className="form" noValidate>
      {state.message ? (
        <p className="notice notice--refused" role="alert">
          {state.message}
        </p>
      ) : notice ? (
        <p className="notice">{notice}</p>
      ) : null}

      <label className="field">
        <span className="field-label">Email</span>
        <input className="input" name="email" type="email" autoComplete="username" defaultValue={state.email} required />
      </label>

      <label className="field">
        <span className="field-label">Password</span>
        <input className="input" name="password" type="password" autoComplete="current-password" required />
      </label>

      <label className="field">
        <span className="field-label">Tenant ID, if you have more than one</span>
        <input className="input mono" name="tenant_id" autoComplete="off" spellCheck={false} />
        <span className="field-hint">The same email can belong to several tenants, each with its own password.</span>
      </label>

      <button className="button" type="submit" disabled={pending}>
        {pending ? 'Signing in' : 'Sign in'}
      </button>
    </form>
  );
}
