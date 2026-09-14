import type { Metadata } from 'next';
import { BlockHead } from '@/components/BlockHead';
import { Tag } from '@/components/Tag';
import { api } from '@/lib/api';
import { current } from '@/lib/current';
import { when } from '@/lib/format';
import { canManageMembers } from '@/lib/roles';
import type { Envelope, WebhookEndpoint } from '@/lib/types';
import { AddEndpointForm } from './AddEndpointForm';
import { ToggleEndpoint } from './ToggleEndpoint';

export const metadata: Metadata = { title: 'Webhooks' };

export default async function WebhooksPage() {
  const { user, tenant } = await current();

  if (!canManageMembers(user.role)) {
    return (
      <div className="page">
        <h1 className="page-title">Webhooks</h1>
        <p className="notice block">
          Managing where {tenant.name} is notified needs the owner or admin role. Your role here is {user.role}.
        </p>
      </div>
    );
  }

  const result = await api<Envelope<WebhookEndpoint[]>>('/api/v1/webhook-endpoints');

  return (
    <div className="page">
      <h1 className="page-title">Webhooks</h1>
      <p className="page-lede">
        Where Keystone tells {tenant.name} that something happened. A notification is recorded in the same transaction as
        the change it describes and sent afterwards, so an address that is down delays the message and never undoes the
        change.
      </p>

      <section className="block" aria-labelledby="endpoints">
        <BlockHead id="endpoints" title="Endpoints" note="Up to 10" />

        {!result.ok ? (
          <p className="notice notice--refused">{result.message}</p>
        ) : result.data.data.length === 0 ? (
          <p className="notice">No endpoint yet. Add one below to be told when a tenant finishes provisioning.</p>
        ) : (
          <div className="ledger-wrap">
            <table className="ledger">
              <thead>
                <tr>
                  <th scope="col">Address</th>
                  <th scope="col">State</th>
                  <th scope="col">Delivered</th>
                  <th scope="col">Waiting</th>
                  <th scope="col">Gave up</th>
                  <th scope="col">Change</th>
                </tr>
              </thead>
              <tbody>
                {result.data.data.map((endpoint) => (
                  <tr key={endpoint.id}>
                    <td>
                      <span className="ledger-name mono">{endpoint.url}</span>
                      <span className="ledger-sub">
                        Added {when(endpoint.created_at)}. Secret ends in <span className="mono">{endpoint.secret_hint}</span>.
                      </span>
                    </td>
                    <td>
                      <Tag tone={endpoint.active ? 'live' : 'plain'}>{endpoint.active ? 'on' : 'off'}</Tag>
                    </td>
                    <td>{endpoint.deliveries.delivered}</td>
                    <td>{endpoint.deliveries.pending}</td>
                    <td>
                      {endpoint.deliveries.dead > 0 ? (
                        <Tag tone="refused">{endpoint.deliveries.dead}</Tag>
                      ) : (
                        endpoint.deliveries.dead
                      )}
                    </td>
                    <td>
                      <ToggleEndpoint id={endpoint.id} active={endpoint.active} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <div className="columns">
        <section className="block" aria-labelledby="add">
          <BlockHead id="add" title="Add an endpoint" />
          <AddEndpointForm />
        </section>

        <aside className="block" aria-labelledby="verify">
          <BlockHead id="verify" title="Checking a delivery is real" />
          <p>
            Every delivery carries a signature: an HMAC-SHA256 of the exact body, made with the endpoint&apos;s secret. The
            receiver computes the same and compares. A request that fails the comparison did not come from Keystone.
          </p>
          <p className="block-foot muted">
            Each delivery also carries the event&apos;s id as its idempotency key. A delivery can arrive more than once, and
            the id is how the receiver notices.
          </p>
        </aside>
      </div>
    </div>
  );
}
