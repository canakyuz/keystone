'use client';

import { useActionState } from 'react';
import { ROLES } from '@/lib/roles';
import type { Role, User } from '@/lib/types';
import { changeRole, setStatus, type ActionState } from './actions';

const idle: ActionState = { tone: null, message: null };

export function MemberControls({ member, callerRole }: { member: User; callerRole: Role }) {
  const [roleState, roleAction, rolePending] = useActionState(changeRole, idle);
  const [statusState, statusAction, statusPending] = useActionState(setStatus, idle);

  if (member.role === 'owner') {
    return <span className="muted">Owners are not changed here.</span>;
  }

  // Only an owner can make an owner. Offering it to anyone else would be offering a
  // button the API refuses.
  const roles = ROLES.filter((role) => role !== 'owner' || callerRole === 'owner');
  const suspended = member.status === 'suspended';
  const latest = statusState.message ? statusState : roleState;

  return (
    <div>
      <div className="row-actions">
        <form action={roleAction} className="inline-form">
          <input type="hidden" name="id" value={member.id} />
          <label className="sr-only" htmlFor={`role-${member.id}`}>
            Role for {member.email}
          </label>
          <select id={`role-${member.id}`} name="role" defaultValue={member.role} className="select is-compact">
            {roles.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
          <button className="button button--quiet button--small" type="submit" disabled={rolePending}>
            Change role
          </button>
        </form>

        <form action={statusAction}>
          <input type="hidden" name="id" value={member.id} />
          <input type="hidden" name="intent" value={suspended ? 'reactivate' : 'suspend'} />
          <button
            className={suspended ? 'button button--quiet button--small' : 'button button--refuse button--small'}
            type="submit"
            disabled={statusPending}
          >
            {suspended ? 'Reactivate' : 'Suspend'}
          </button>
        </form>
      </div>

      {latest.message ? (
        <p className={`row-message is-${latest.tone}`} role="status">
          {latest.message}
        </p>
      ) : null}
    </div>
  );
}
