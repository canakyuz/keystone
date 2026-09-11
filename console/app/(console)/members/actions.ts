'use server';

import { revalidatePath } from 'next/cache';
import { api, type ApiResult } from '@/lib/api';
import { UUID } from '@/lib/format';

export interface ActionState {
  tone: 'done' | 'refused' | null;
  message: string | null;
}

export async function changeRole(_: ActionState, form: FormData): Promise<ActionState> {
  const id = String(form.get('id') ?? '');
  const role = String(form.get('role') ?? '');
  if (!UUID.test(id)) return refused('That member could not be identified.');

  const result = await api(`/api/v1/users/${id}/role`, { method: 'POST', body: { role } });

  return settle(result, `Role changed to ${role}.`);
}

export async function setStatus(_: ActionState, form: FormData): Promise<ActionState> {
  const id = String(form.get('id') ?? '');
  const intent = String(form.get('intent') ?? '');
  if (!UUID.test(id)) return refused('That member could not be identified.');

  if (intent === 'suspend') {
    const result = await api(`/api/v1/users/${id}/suspend`, {
      method: 'POST',
      body: { reason: 'Suspended from the Keystone console' },
    });
    return settle(result, 'Suspended. Their next request will be refused.');
  }

  const result = await api(`/api/v1/users/${id}/activate`, { method: 'POST', body: {} });

  return settle(result, 'Reactivated.');
}

export async function addMember(_: ActionState, form: FormData): Promise<ActionState> {
  const body = {
    email: String(form.get('email') ?? '').trim(),
    first_name: String(form.get('first_name') ?? '').trim(),
    last_name: String(form.get('last_name') ?? '').trim(),
    role: String(form.get('role') ?? 'viewer'),
    password: String(form.get('password') ?? ''),
  };

  if (!body.email || !body.first_name || !body.last_name) return refused('Email, first name and last name are all needed.');
  if (body.password.length < 8) return refused('The first password needs at least 8 characters.');

  const result = await api('/api/v1/users', { method: 'POST', body });

  return settle(result, `Added ${body.email} as ${body.role}.`);
}

function settle(result: ApiResult<unknown>, done: string): ActionState {
  if (result.ok) {
    revalidatePath('/members');
    revalidatePath('/');
    return { tone: 'done', message: done };
  }

  return refused(result.status === 403 ? `Refused by the API. ${result.message}` : result.message);
}

function refused(message: string): ActionState {
  return { tone: 'refused', message };
}
