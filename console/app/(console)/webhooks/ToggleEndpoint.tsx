'use client';

import { useActionState } from 'react';
import { setActive, type ToggleState } from './actions';

const idle: ToggleState = { message: null };

export function ToggleEndpoint({ id, active }: { id: string; active: boolean }) {
  const [state, action, pending] = useActionState(setActive, idle);

  return (
    <form action={action}>
      <input type="hidden" name="id" value={id} />
      <input type="hidden" name="active" value={active ? 'false' : 'true'} />
      <button
        className={active ? 'button button--refuse button--small' : 'button button--quiet button--small'}
        type="submit"
        disabled={pending}
      >
        {active ? 'Turn off' : 'Turn on'}
      </button>
      {state.message ? (
        <p className="row-message is-refused" role="status">
          {state.message}
        </p>
      ) : null}
    </form>
  );
}
