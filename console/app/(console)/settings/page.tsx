import type { Metadata } from 'next';
import { BlockHead } from '@/components/BlockHead';
import { current } from '@/lib/current';
import { canManageMembers } from '@/lib/roles';
import { SettingsForm } from './SettingsForm';

export const metadata: Metadata = { title: 'Settings' };

export default async function SettingsPage() {
  const { user, tenant } = await current();

  return (
    <div className="page">
      <h1 className="page-title">Settings</h1>
      <p className="page-lede">
        What {tenant.name} is called and where to reach it. The slug and the schema are not here: they name the
        tenant&apos;s data and cannot change once it exists.
      </p>

      <div className="columns">
        <section className="block" aria-labelledby="details">
          <BlockHead id="details" title="Tenant details" />
          {canManageMembers(user.role) ? (
            <SettingsForm tenant={tenant} />
          ) : (
            <p className="notice">
              Changing these needs the owner or admin role. Your role in {tenant.name} is {user.role}.
            </p>
          )}
        </section>

        <aside className="block" aria-labelledby="fixed">
          <BlockHead id="fixed" title="What cannot change here" />
          <dl className="kv">
            <div className="kv-row">
              <dt>Slug</dt>
              <dd className="mono">{tenant.slug}</dd>
            </div>
            <div className="kv-row">
              <dt>Schema</dt>
              <dd className="mono">{tenant.schema_name}</dd>
            </div>
            <div className="kv-row">
              <dt>Plan</dt>
              <dd>{tenant.plan}</dd>
            </div>
            <div className="kv-row">
              <dt>Status</dt>
              <dd>{tenant.status}</dd>
            </div>
          </dl>
          <p className="block-foot muted">
            The plan and the status are changed by an operator, not from inside the tenant. Every change to either one
            is written to the history.
          </p>
        </aside>
      </div>
    </div>
  );
}
