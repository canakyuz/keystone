import type { Metadata } from 'next';
import Link from 'next/link';
import { BlockHead } from '@/components/BlockHead';
import { Tag } from '@/components/Tag';
import { api } from '@/lib/api';
import { current } from '@/lib/current';
import { when } from '@/lib/format';
import { canManageMembers } from '@/lib/roles';
import type { AuditEntry, AuditPage, Envelope } from '@/lib/types';

export const metadata: Metadata = { title: 'History' };

// What each action did, in the words a reader of the history would use.
const ACTIONS: Record<string, string> = {
  'user.role_changed': 'Role changed',
  'user.created': 'Member added',
  'user.password_changed': 'Password changed',
  'user.profile_updated': 'Profile updated',
  'user.suspended': 'Member suspended',
  'user.reactivated': 'Member reactivated',
  'user.deleted': 'Member removed',
  'tenant.suspended': 'Tenant suspended',
  'tenant.reactivated': 'Tenant reactivated',
  'tenant.plan_changed': 'Plan changed',
  'tenant.updated': 'Details updated',
  'tenant.branding_updated': 'Branding updated',
  'tenant.domain_set': 'Custom domain set',
  'tenant.domain_verified': 'Custom domain verified',
  'tenant.deleted': 'Tenant removed',
  'tenant.activated': 'Tenant provisioned',
};

export default async function AuditPage({ searchParams }: { searchParams: Promise<{ before?: string }> }) {
  const { user, tenant } = await current();
  const { before } = await searchParams;

  if (!canManageMembers(user.role)) {
    return (
      <div className="page">
        <h1 className="page-title">History</h1>
        <p className="notice block">
          Reading the history needs the owner or admin role. Your role in {tenant.name} is {user.role}. A member who was
          suspended would want to read who suspended them, which is why this is not open to everyone.
        </p>
      </div>
    );
  }

  const query = new URLSearchParams({ limit: '50' });
  if (before) query.set('before', before);

  const result = await api<Envelope<AuditPage>>(`/api/v1/audit?${query.toString()}`);

  return (
    <div className="page">
      <h1 className="page-title">History</h1>
      <p className="page-lede">
        What changed in {tenant.name}, who changed it, and when. Each line was written in the same transaction as the
        change it describes, and the table it lives in accepts inserts and nothing else.
      </p>

      <section className="block" aria-labelledby="trail">
        <BlockHead id="trail" title={before ? 'Older changes' : 'Most recent first'} />

        {!result.ok ? (
          <p className="notice notice--refused">{result.message}</p>
        ) : result.data.data.entries.length === 0 ? (
          <p className="notice">
            Nothing here yet. Changing a member&apos;s role or suspending an account writes the first line.
          </p>
        ) : (
          <>
            <div className="ledger-wrap">
              <table className="ledger">
                <thead>
                  <tr>
                    <th scope="col">When</th>
                    <th scope="col">What</th>
                    <th scope="col">Who</th>
                    <th scope="col">Details</th>
                  </tr>
                </thead>
                <tbody>
                  {result.data.data.entries.map((entry) => (
                    <tr key={entry.id}>
                      <td className="muted">{when(entry.created_at)}</td>
                      <td>
                        <span className="ledger-name">{ACTIONS[entry.action] ?? entry.action}</span>
                        <span className="ledger-sub mono">{entry.subject_type}</span>
                      </td>
                      <td>{actor(entry)}</td>
                      <td>{details(entry)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="row-actions block">
              {before ? (
                <Link className="button button--quiet button--small" href="/audit">
                  Back to the newest
                </Link>
              ) : null}
              {result.data.data.next_before ? (
                <Link
                  className="button button--quiet button--small"
                  href={`/audit?before=${encodeURIComponent(result.data.data.next_before)}`}
                >
                  Older
                </Link>
              ) : (
                <span className="muted">That is the whole history.</span>
              )}
            </div>
          </>
        )}
      </section>
    </div>
  );
}

function actor(entry: AuditEntry) {
  if (entry.actor_type !== 'user') {
    return <Tag>{entry.actor_type}</Tag>;
  }

  return <span>{entry.actor_email || <span className="mono">{entry.actor_id}</span>}</span>;
}

// The metadata is written so an entry can be read without looking anything up, which is the
// point of showing it rather than a link to a row that may since have changed.
function details(entry: AuditEntry) {
  const pairs = Object.entries(entry.metadata).filter(([, value]) => value !== null && value !== '');

  if (pairs.length === 0) {
    return <span className="muted">None</span>;
  }

  return (
    <span className="mono">
      {pairs.map(([key, value]) => `${key}: ${String(value)}`).join(', ')}
    </span>
  );
}
