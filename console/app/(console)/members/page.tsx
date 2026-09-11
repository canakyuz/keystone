import type { Metadata } from 'next';
import { BlockHead } from '@/components/BlockHead';
import { StatusTag, Tag } from '@/components/Tag';
import { api } from '@/lib/api';
import { current } from '@/lib/current';
import { when } from '@/lib/format';
import { canManageMembers } from '@/lib/roles';
import type { Envelope, UserList } from '@/lib/types';
import { AddMemberForm } from './AddMemberForm';
import { MemberControls } from './MemberControls';

export const metadata: Metadata = { title: 'Members' };

export default async function MembersPage({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const { user, tenant } = await current();
  const { q = '' } = await searchParams;

  const query = new URLSearchParams({ per_page: '100' });
  if (q.trim()) query.set('search', q.trim());

  const list = await api<Envelope<UserList>>(`/api/v1/users?${query.toString()}`);
  const manage = canManageMembers(user.role);

  return (
    <div className="page">
      <h1 className="page-title">Members</h1>
      <p className="page-lede">
        Everyone who can sign in to {tenant.name}. Each acts with the role recorded here, and a change takes effect on their
        next request rather than when their token expires.
      </p>

      <form className="inline-form block" role="search" action="/members">
        <label className="sr-only" htmlFor="member-search">
          Search members
        </label>
        <input id="member-search" className="input" name="q" defaultValue={q} placeholder="Name or email" />
        <button className="button button--quiet" type="submit">
          Search
        </button>
      </form>

      <section className="block" aria-labelledby="member-list">
        <BlockHead id="member-list" title={q ? `Matching "${q}"` : 'Everyone'} note={list.ok ? `${list.data.data.total} found` : undefined} />

        {!list.ok ? (
          <p className="notice notice--refused">{list.message}</p>
        ) : list.data.data.data.length === 0 ? (
          <p className="notice">No member matches that. Search by part of a name or an email address.</p>
        ) : (
          <div className="ledger-wrap">
            <table className="ledger">
              <thead>
                <tr>
                  <th scope="col">Member</th>
                  <th scope="col">Role</th>
                  <th scope="col">Status</th>
                  <th scope="col">Last sign-in</th>
                  {manage ? <th scope="col">Change</th> : null}
                </tr>
              </thead>
              <tbody>
                {list.data.data.data.map((member) => (
                  <tr key={member.id}>
                    <td>
                      <span className="ledger-name">
                        {member.full_name}
                        {member.id === user.id ? <span className="muted"> (you)</span> : null}
                      </span>
                      <span className="ledger-sub">{member.email}</span>
                    </td>
                    <td>
                      <Tag tone={member.role === 'owner' || member.role === 'admin' ? 'accent' : 'plain'}>{member.role}</Tag>
                    </td>
                    <td>
                      <StatusTag status={member.status} />
                    </td>
                    <td className="muted">{when(member.last_login_at)}</td>
                    {manage ? (
                      <td>
                        <MemberControls member={member} callerRole={user.role} />
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="block" aria-labelledby="add-member">
        <BlockHead id="add-member" title="Add a member" />
        {manage ? (
          <AddMemberForm callerRole={user.role} />
        ) : (
          <p className="notice">
            Adding members, changing roles and suspending accounts need the owner or admin role. Your role in {tenant.name}{' '}
            is {user.role}.
          </p>
        )}
      </section>
    </div>
  );
}
