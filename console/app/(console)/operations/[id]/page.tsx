import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { AutoRefresh } from '@/components/AutoRefresh';
import { BlockHead } from '@/components/BlockHead';
import { StatusTag } from '@/components/Tag';
import { api } from '@/lib/api';
import { current } from '@/lib/current';
import { took, UUID, when } from '@/lib/format';
import type { Operation } from '@/lib/types';

export const metadata: Metadata = { title: 'Operation' };

const HEADLINES: Record<Operation['status'], string> = {
  pending: 'Waiting for a worker',
  running: 'Provisioning',
  succeeded: 'The tenant is active',
  failed: 'Provisioning failed',
};

const LEDES: Record<Operation['status'], string> = {
  pending: 'The request is recorded and a job is queued. If this does not move, no worker is running.',
  running: 'A worker holds the job under a lease and is creating the schema. If it dies, another takes the job over when the lease runs out.',
  succeeded: 'The schema exists and the tenant accepts requests. Its activation, its audit entry and its notification were written together.',
  failed: 'The job ran out of attempts. The tenant was not activated.',
};

export default async function OperationPage({ params }: { params: Promise<{ id: string }> }) {
  await current();
  const { id } = await params;
  if (!UUID.test(id)) notFound();

  const result = await api<Operation>(`/api/v1/operations/${id}`);
  if (!result.ok && result.status === 404) notFound();

  if (!result.ok) {
    return (
      <div className="page">
        <h1 className="page-title">The operation could not be read</h1>
        <p className="notice notice--refused block">{result.message}</p>
      </div>
    );
  }

  const op = result.data;
  const terminal = op.status === 'succeeded' || op.status === 'failed';
  const position = op.status === 'pending' ? 0 : op.status === 'running' ? 1 : 2;

  const steps = [
    { name: 'Accepted', note: 'Recorded, with a job queued' },
    { name: 'Provisioning', note: 'A worker holds the job' },
    op.status === 'failed'
      ? { name: 'Failed', note: 'Attempts ran out' }
      : { name: 'Active', note: 'The tenant serves requests' },
  ];

  return (
    <div className="page">
      <p className="muted">
        Operation <span className="mono">{op.id}</span>
      </p>
      <h1 className="page-title">{HEADLINES[op.status]}</h1>
      <p className="page-lede">{LEDES[op.status]}</p>

      <ol className="track block" aria-label="Progress">
        {steps.map((step, index) => {
          const state =
            index < position ? 'is-past' : index === position ? (op.status === 'failed' ? 'is-failed' : 'is-now') : '';

          return (
            <li key={step.name} className={`track-step ${state}`} aria-current={index === position ? 'step' : undefined}>
              <p className="track-step-name">{step.name}</p>
              <p className="track-step-note">{step.note}</p>
            </li>
          );
        })}
      </ol>

      {terminal ? null : <AutoRefresh everyMs={1500} />}

      <section className="block" aria-labelledby="details">
        <BlockHead id="details" title="The operation's record" />
        <dl className="kv">
          <div className="kv-row">
            <dt>Status</dt>
            <dd>
              <StatusTag status={op.status} />
            </dd>
          </div>
          <div className="kv-row">
            <dt>Tenant ID</dt>
            <dd className="mono">{op.tenant_id}</dd>
          </div>
          <div className="kv-row">
            <dt>Kind</dt>
            <dd className="mono">{op.kind}</dd>
          </div>
          <div className="kv-row">
            <dt>Accepted</dt>
            <dd>{when(op.created_at)}</dd>
          </div>
          <div className="kv-row">
            <dt>{terminal ? 'Took' : 'Waiting for'}</dt>
            <dd>{took(op.created_at, op.completed_at)}</dd>
          </div>
          {op.error_code ? (
            <div className="kv-row">
              <dt>Error</dt>
              <dd>
                <span className="mono">{op.error_code}</span>
                {op.error_detail ? <span className="ledger-sub">{op.error_detail}</span> : null}
              </dd>
            </div>
          ) : null}
        </dl>
      </section>
    </div>
  );
}
