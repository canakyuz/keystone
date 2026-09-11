import type { Metadata } from 'next';
import { BlockHead } from '@/components/BlockHead';
import { StatusTag, Tag } from '@/components/Tag';
import { api, probe } from '@/lib/api';
import { current } from '@/lib/current';
import { bytes, capitalise, clock, limit, when } from '@/lib/format';
import type { Envelope } from '@/lib/types';

export const metadata: Metadata = { title: 'Tenant' };

const ROLE_ORDER = ['owner', 'admin', 'editor', 'viewer'] as const;
const STATUS_ORDER = ['active', 'pending', 'suspended', 'inactive'] as const;

export default async function TenantPage() {
  const { user, tenant } = await current();

  const [health, ready, stats] = await Promise.all([
    probe('/health'),
    probe('/ready'),
    api<Envelope<Record<string, number>>>('/api/v1/users/stats'),
  ]);

  const counts = stats.ok ? stats.data.data : null;
  const measuredAt = clock(new Date());

  return (
    <>
      <section className="masthead" aria-labelledby="tenant-name">
        <h1 id="tenant-name" className="masthead-name">
          {tenant.name}
        </h1>
        <p className="masthead-line">
          {capitalise(tenant.plan)} plan. Its data lives in its own schema, <span className="mono">{tenant.schema_name}</span>.
        </p>
      </section>

      <div className="ticker" aria-label={`Measured from this server at ${measuredAt}`}>
        <span className={health.up ? 'ticker-cell' : 'ticker-cell is-down'}>
          API <strong>{health.up ? `${health.ms} ms` : 'not answering'}</strong>
        </span>
        <span className={ready.up ? 'ticker-cell' : 'ticker-cell is-down'}>
          Database and Redis <strong>{ready.up ? 'ready' : 'not ready'}</strong>
        </span>
        <span className="ticker-cell">
          Tenant <strong>{tenant.status}</strong>
        </span>
        <span className="ticker-cell">
          Members <strong>{counts ? counts.total : 'unknown'}</strong>
        </span>
        <span className="ticker-cell">
          Measured at <strong>{measuredAt}</strong>
        </span>
      </div>

      <div className="page">
        <div className="columns">
          <section className="block" aria-labelledby="record">
            <BlockHead id="record" title="The tenant's record" />
            <dl className="kv">
              <Row term="ID">
                <span className="mono">{tenant.id}</span>
              </Row>
              <Row term="Slug">
                <span className="mono">{tenant.slug}</span>
              </Row>
              <Row term="Status">
                <StatusTag status={tenant.status} />
              </Row>
              <Row term="Contact">{tenant.email}</Row>
              <Row term="Created">{when(tenant.created_at)}</Row>
              {tenant.trial_ends_at ? <Row term="Trial ends">{when(tenant.trial_ends_at)}</Row> : null}
              <Row term="Custom domain">{tenant.custom_domain || 'None'}</Row>
            </dl>
          </section>

          <section className="block" aria-labelledby="you">
            <BlockHead id="you" title="Your membership" />
            <dl className="kv">
              <Row term="Name">{user.full_name}</Row>
              <Row term="Role">
                <Tag tone="accent">{user.role}</Tag>
              </Row>
              <Row term="Status">
                <StatusTag status={user.status} />
              </Row>
              <Row term="Last sign-in">{when(user.last_login_at)}</Row>
            </dl>
            <p className="block-foot muted">
              Read from this tenant&apos;s record on every request, not from your token. If an administrator suspends you
              or changes your role, it applies to your next click.
            </p>
          </section>
        </div>

        <div className="columns">
          <section className="block" aria-labelledby="members">
            <BlockHead id="members" title="Members" note={counts ? `${counts.total} in all` : undefined} />
            {counts ? (
              <div className="count-grid">
                <dl className="kv">
                  {ROLE_ORDER.map((role) => (
                    <Row key={role} term={capitalise(role)}>
                      {counts[role] ?? 0}
                    </Row>
                  ))}
                </dl>
                <dl className="kv">
                  {STATUS_ORDER.map((status) => (
                    <Row key={status} term={capitalise(status)}>
                      {counts[status] ?? 0}
                    </Row>
                  ))}
                </dl>
              </div>
            ) : (
              <p className="notice notice--refused">{stats.ok ? null : stats.message}</p>
            )}
          </section>

          <section className="block" aria-labelledby="limits">
            <BlockHead id="limits" title="What the plan allows" />
            <dl className="kv">
              <Row term="Members">{limit(tenant.feature_limits.max_users)}</Row>
              <Row term="Websites">{limit(tenant.feature_limits.max_websites)}</Row>
              <Row term="Storage">{bytes(tenant.feature_limits.max_storage)}</Row>
              <Row term="Custom domain">{tenant.feature_limits.custom_domain ? 'Yes' : 'No'}</Row>
              <Row term="API access">{tenant.feature_limits.api_access ? 'Yes' : 'No'}</Row>
            </dl>
          </section>
        </div>
      </div>
    </>
  );
}

function Row({ term, children }: { term: string; children: React.ReactNode }) {
  return (
    <div className="kv-row">
      <dt>{term}</dt>
      <dd>{children}</dd>
    </div>
  );
}
