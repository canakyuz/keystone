'use server';

import { revalidatePath } from 'next/cache';
import { api } from '@/lib/api';
import { UUID } from '@/lib/format';

export interface SettingsState {
  tone: 'done' | 'refused' | null;
  message: string | null;
}

export async function updateTenant(_: SettingsState, form: FormData): Promise<SettingsState> {
  const id = String(form.get('id') ?? '');
  const name = String(form.get('name') ?? '').trim();
  const email = String(form.get('email') ?? '').trim();
  const phone = String(form.get('phone') ?? '').trim();

  if (!UUID.test(id)) return { tone: 'refused', message: 'That tenant could not be identified.' };
  if (name.length < 2) return { tone: 'refused', message: 'A name needs at least two characters.' };
  if (!email) return { tone: 'refused', message: 'A contact email is needed.' };

  const result = await api(`/api/v1/tenants/${id}`, {
    method: 'PATCH',
    body: { name, email, ...(phone ? { phone } : {}) },
  });

  if (!result.ok) {
    return {
      tone: 'refused',
      message: result.status === 403 ? `Refused by the API. ${result.message}` : result.message,
    };
  }

  revalidatePath('/settings');
  revalidatePath('/');

  return { tone: 'done', message: 'Saved.' };
}
