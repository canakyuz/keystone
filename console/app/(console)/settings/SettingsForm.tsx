'use client';

import { useActionState } from 'react';
import type { Tenant } from '@/lib/types';
import { updateTenant, type SettingsState } from './actions';

const idle: SettingsState = { tone: null, message: null };

export function SettingsForm({ tenant }: { tenant: Tenant }) {
  const [state, action, pending] = useActionState(updateTenant, idle);

  return (
    <form action={action} className="form">
      <input type="hidden" name="id" value={tenant.id} />

      {state.message ? (
        <p className={`notice notice--${state.tone}`} role="status">
          {state.message}
        </p>
      ) : null}

      <label className="field">
        <span className="field-label">Name</span>
        <input className="input" name="name" defaultValue={tenant.name} required minLength={2} />
        <span className="field-hint">Shown at the top of the console and on anything the tenant sends.</span>
      </label>

      <label className="field">
        <span className="field-label">Contact email</span>
        <input className="input" name="email" type="email" defaultValue={tenant.email} required />
      </label>

      <label className="field">
        <span className="field-label">Phone</span>
        <input className="input" name="phone" defaultValue={tenant.phone ?? ''} />
      </label>

      <button className="button" type="submit" disabled={pending}>
        {pending ? 'Saving…' : 'Save changes'}
      </button>
    </form>
  );
}
