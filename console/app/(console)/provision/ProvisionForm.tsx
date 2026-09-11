'use client';

import { useActionState } from 'react';
import { provisionTenant, type ProvisionState } from './actions';

const initial: ProvisionState = { message: null, values: { name: '', slug: '', email: '', plan: 'free' } };
const PLANS = ['free', 'starter', 'pro', 'enterprise'];

export function ProvisionForm({ idempotencyKey }: { idempotencyKey: string }) {
  const [state, action, pending] = useActionState(provisionTenant, initial);

  return (
    <form action={action} className="form">
      <input type="hidden" name="idempotency_key" value={idempotencyKey} />

      {state.message ? (
        <p className="notice notice--refused" role="alert">
          {state.message}
        </p>
      ) : null}

      <label className="field">
        <span className="field-label">Name</span>
        <input className="input" name="name" defaultValue={state.values.name} autoComplete="organization" required />
      </label>

      <div className="form-row">
        <label className="field">
          <span className="field-label">Slug</span>
          <input
            className="input mono"
            name="slug"
            defaultValue={state.values.slug}
            autoComplete="off"
            spellCheck={false}
            required
          />
          <span className="field-hint">Names the schema. It cannot be changed later.</span>
        </label>
        <label className="field">
          <span className="field-label">Plan</span>
          <select className="select" name="plan" defaultValue={state.values.plan}>
            {PLANS.map((plan) => (
              <option key={plan} value={plan}>
                {plan}
              </option>
            ))}
          </select>
        </label>
      </div>

      <label className="field">
        <span className="field-label">Contact email</span>
        <input className="input" name="email" type="email" defaultValue={state.values.email} autoComplete="off" required />
      </label>

      <button className="button" type="submit" disabled={pending}>
        {pending ? 'Sending the request' : 'Create tenant'}
      </button>
    </form>
  );
}
