'use server';

import { revalidatePath } from 'next/cache';
import { api } from '@/lib/api';
import { UUID } from '@/lib/format';
import type { CreatedWebhookEndpoint, Envelope } from '@/lib/types';

export interface AddState {
  tone: 'done' | 'refused' | null;
  message: string | null;
  // The signing secret, present exactly once: in the state returned by the request that
  // created the endpoint. Nothing stores it, so a reload loses it, which is the point.
  secret: string | null;
  url: string;
}

export async function addEndpoint(_: AddState, form: FormData): Promise<AddState> {
  const url = String(form.get('url') ?? '').trim();
  if (!url) return { tone: 'refused', message: 'Enter the https address to notify.', secret: null, url };

  const result = await api<Envelope<CreatedWebhookEndpoint>>('/api/v1/webhook-endpoints', {
    method: 'POST',
    body: { url },
  });

  if (!result.ok) {
    const message = result.status === 403 ? `Refused by the API. ${result.message}` : result.message;
    return { tone: 'refused', message, secret: null, url };
  }

  revalidatePath('/webhooks');

  return { tone: 'done', message: null, secret: result.data.data.secret, url: '' };
}

export interface ToggleState {
  message: string | null;
}

export async function setActive(_: ToggleState, form: FormData): Promise<ToggleState> {
  const id = String(form.get('id') ?? '');
  const active = form.get('active') === 'true';
  if (!UUID.test(id)) return { message: 'That endpoint could not be identified.' };

  const result = await api(`/api/v1/webhook-endpoints/${id}`, { method: 'PATCH', body: { active } });
  if (!result.ok) return { message: result.message };

  revalidatePath('/webhooks');

  return { message: null };
}
