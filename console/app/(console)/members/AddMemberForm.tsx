'use client';

import { useActionState } from 'react';
import { ROLES } from '@/lib/roles';
import type { Role } from '@/lib/types';
import { addMember, type ActionState } from './actions';

const idle: ActionState = { tone: null, message: null };

export function AddMemberForm({ callerRole }: { callerRole: Role }) {
  const [state, action, pending] = useActionState(addMember, idle);
  const roles = ROLES.filter((role) => role !== 'owner' || callerRole === 'owner');

  return (
    <form action={action} className="form">
      {state.message ? (
        <p className={`notice notice--${state.tone}`} role="status">
          {state.message}
        </p>
      ) : null}

      <label className="field">
        <span className="field-label">Email</span>
        <input className="input" name="email" type="email" autoComplete="off" required />
      </label>

      <div className="form-row">
        <label className="field">
          <span className="field-label">First name</span>
          <input className="input" name="first_name" autoComplete="off" required />
        </label>
        <label className="field">
          <span className="field-label">Last name</span>
          <input className="input" name="last_name" autoComplete="off" required />
        </label>
      </div>

      <div className="form-row">
        <label className="field">
          <span className="field-label">Role</span>
          <select className="select" name="role" defaultValue="viewer">
            {roles.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          <span className="field-label">First password</span>
          <input className="input" name="password" type="password" autoComplete="new-password" minLength={8} required />
        </label>
      </div>

      <button className="button" type="submit" disabled={pending}>
        {pending ? 'Adding member' : 'Add member'}
      </button>
    </form>
  );
}
