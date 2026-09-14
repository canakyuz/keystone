'use client';

import { useActionState } from 'react';
import { addEndpoint, type AddState } from './actions';

const idle: AddState = { tone: null, message: null, secret: null, url: '' };

export function AddEndpointForm() {
  const [state, action, pending] = useActionState(addEndpoint, idle);

  return (
    <div>
      {state.secret ? (
        <div className="notice notice--done block-gap" role="status">
          <p>
            Added. This is the signing secret. Copy it now: it is shown once and cannot be read again, only replaced by
            adding the endpoint anew.
          </p>
          <p className="secret mono">{state.secret}</p>
        </div>
      ) : null}

      <form action={action} className="form">
        {state.tone === 'refused' && state.message ? (
          <p className="notice notice--refused" role="alert">
            {state.message}
          </p>
        ) : null}

        <label className="field">
          <span className="field-label">Address to notify</span>
          <input
            className="input mono"
            name="url"
            type="url"
            inputMode="url"
            placeholder="https://example.com/keystone"
            defaultValue={state.url}
            autoComplete="off"
            spellCheck={false}
            required
          />
          <span className="field-hint">
            https only. An address inside a private network is refused here, and again when a delivery is sent.
          </span>
        </label>

        <button className="button" type="submit" disabled={pending}>
          {pending ? 'Adding…' : 'Add endpoint'}
        </button>
      </form>
    </div>
  );
}
