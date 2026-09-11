import 'server-only';
import { cache } from 'react';
import { redirect } from 'next/navigation';
import { api } from './api';
import { readToken } from './session';
import type { Envelope, Tenant, User } from './types';

export interface Current {
  user: User;
  tenant: Tenant;
}

/**
 * Who is signed in, and into which tenant, read from the API once per request.
 *
 * The role this console shows and acts on comes from here, which is to say from the
 * tenant's own record and not from the token. A 401 means the token expired; a 403 means
 * the API no longer accepts the membership behind it. Either way the cookie is stale.
 */
export const current = cache(async (): Promise<Current> => {
  if (!(await readToken())) redirect('/login');

  const [me, tenant] = await Promise.all([
    api<Envelope<User>>('/api/v1/auth/me'),
    api<Envelope<Tenant>>('/api/v1/tenants/current'),
  ]);

  if (!me.ok && me.status === 401) redirect('/logout?reason=expired');
  if (!me.ok && me.status === 403) redirect('/logout?reason=revoked');
  if (!me.ok) throw new Error(me.message);
  if (!tenant.ok) throw new Error(tenant.message);

  return { user: me.data.data, tenant: tenant.data.data };
});
