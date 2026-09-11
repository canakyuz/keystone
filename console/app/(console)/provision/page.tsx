import type { Metadata } from 'next';
import { BlockHead } from '@/components/BlockHead';
import { current } from '@/lib/current';
import { ProvisionForm } from './ProvisionForm';

export const metadata: Metadata = { title: 'New tenant' };

export default async function ProvisionPage() {
  await current();

  // One key per visit to this page. A double click or a retry after a timeout sends the
  // same key again, and the API answers with the operation it already started.
  const idempotencyKey = crypto.randomUUID();

  return (
    <div className="page">
      <h1 className="page-title">New tenant</h1>
      <p className="page-lede">
        A tenant is not created while you wait. The API records the request and answers at once; a separate worker creates
        the schema and activates the tenant, and the next page follows it.
      </p>

      <div className="columns">
        <section className="block" aria-label="Tenant details">
          <ProvisionForm idempotencyKey={idempotencyKey} />
        </section>

        <aside className="block" aria-labelledby="twice">
          <BlockHead id="twice" title="If it is sent twice" />
          <p>
            This form carries one idempotency key, <span className="mono">{idempotencyKey}</span>. Sending it again returns
            the same operation instead of starting a second tenant. Sending the same key with different details is refused.
          </p>
          <p className="block-foot muted">
            You will not become a member of the new tenant. Keystone does not add the person who asked for it.
          </p>
        </aside>
      </div>
    </div>
  );
}
