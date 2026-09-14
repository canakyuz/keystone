'use client';

import { useActionState, useState } from 'react';
import { signIn, type SignInState } from './actions';

const initial: SignInState = { message: null, email: '', tenantId: '' };

export function LoginForm({ notice }: { notice: string | null }) {
  const [state, action, pending] = useActionState(signIn, initial);
  const [revealed, setRevealed] = useState(false);
  const refused = Boolean(state.message);

  return (
    <form action={action} className="form" noValidate aria-busy={pending}>
      {refused ? (
        <p id="signin-error" className="notice notice--refused" role="alert">
          {state.message}
        </p>
      ) : notice ? (
        <p className="notice" role="status">
          {notice}
        </p>
      ) : null}

      <div className="field">
        <label className="field-label" htmlFor="signin-email">
          Email
        </label>
        <input
          id="signin-email"
          className="input"
          name="email"
          type="email"
          autoComplete="username"
          defaultValue={state.email}
          aria-invalid={refused || undefined}
          aria-describedby={refused ? 'signin-error' : undefined}
          required
        />
      </div>

      <div className="field">
        <label className="field-label" htmlFor="signin-password">
          Password
        </label>
        <div className="field-control">
          <input
            id="signin-password"
            className="input"
            name="password"
            type={revealed ? 'text' : 'password'}
            autoComplete="current-password"
            aria-invalid={refused || undefined}
            aria-describedby={refused ? 'signin-error' : undefined}
            required
          />
          <button
            type="button"
            className="field-reveal"
            aria-controls="signin-password"
            aria-pressed={revealed}
            aria-label={revealed ? 'Hide password' : 'Show password'}
            onClick={() => setRevealed((r) => !r)}
          >
            {revealed ? 'Hide' : 'Show'}
          </button>
        </div>
      </div>

      {/* Most accounts belong to one tenant, so the field waits behind a disclosure; it stays
          open after a refused attempt that carried a tenant, so the value is not hidden. */}
      <details className="disclosure" open={state.tenantId ? true : undefined}>
        <summary>I belong to more than one tenant</summary>
        <div className="field">
          <label className="field-label" htmlFor="signin-tenant">
            Tenant ID
          </label>
          <input
            id="signin-tenant"
            className="input mono"
            name="tenant_id"
            autoComplete="off"
            spellCheck={false}
            defaultValue={state.tenantId}
            aria-describedby="signin-tenant-hint"
          />
          <span id="signin-tenant-hint" className="field-hint">
            The same email can belong to several tenants, each with its own password.
          </span>
        </div>
      </details>

      <button className="button" type="submit" disabled={pending}>
        {pending ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  );
}
