'use server';

import { redirect } from 'next/navigation';
import { api } from '@/lib/api';
import type { Operation } from '@/lib/types';

export interface ProvisionState {
  message: string | null;
  values: { name: string; slug: string; email: string; plan: string };
}

const SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export async function provisionTenant(_: ProvisionState, form: FormData): Promise<ProvisionState> {
  const values = {
    name: String(form.get('name') ?? '').trim(),
    slug: String(form.get('slug') ?? '').trim(),
    email: String(form.get('email') ?? '').trim(),
    plan: String(form.get('plan') ?? 'free'),
  };
  const key = String(form.get('idempotency_key') ?? '');

  if (!values.name || !values.slug || !values.email) {
    return { message: 'A name, a slug and a contact email are all needed.', values };
  }
  if (!SLUG.test(values.slug)) {
    return { message: 'Use lowercase letters and digits for the slug, with single hyphens between words.', values };
  }

  const result = await api<{ operation: Operation; tenant_id: string }>('/api/v1/tenants', {
    method: 'POST',
    body: values,
    headers: key ? { 'Idempotency-Key': key } : {},
  });

  if (!result.ok) {
    if (result.code === 'slug_taken') return { message: `The slug "${values.slug}" is taken. Choose another.`, values };
    if (result.code === 'idempotency_key_reused') {
      return {
        message: 'These details differ from the first time this form was sent. Reload the page to start a new request.',
        values,
      };
    }
    return { message: result.status === 403 ? `Refused by the API. ${result.message}` : result.message, values };
  }

  redirect(`/operations/${result.data.operation.id}`);
}
