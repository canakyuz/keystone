import 'server-only';
import { API_URL } from './config';
import { readToken } from './session';

export type ApiResult<T> =
  | { ok: true; status: number; data: T }
  | { ok: false; status: number; message: string; code?: string };

interface CallOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  body?: unknown;
  headers?: Record<string, string>;
  /** Send the session token. Only sign-in calls the API without one. */
  authenticated?: boolean;
}

const TIMEOUT_MS = 8_000;

/**
 * Calls the Keystone API from the server.
 *
 * It never throws for an HTTP answer. A 403 is information a page shows, not an exception
 * to recover from. A status of 0 means the API could not be reached at all, so a page can
 * tell "the API said no" from "the API is not there".
 */
export async function api<T>(path: string, options: CallOptions = {}): Promise<ApiResult<T>> {
  const headers: Record<string, string> = { Accept: 'application/json', ...options.headers };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';

  if (options.authenticated !== false) {
    const token = await readToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }

  let response: Response;
  try {
    response = await fetch(`${API_URL}${path}`, {
      method: options.method ?? 'GET',
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      cache: 'no-store',
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
  } catch {
    return { ok: false, status: 0, message: `The Keystone API did not answer at ${API_URL}.` };
  }

  const payload: unknown = await response.json().catch(() => null);

  if (response.ok) return { ok: true, status: response.status, data: payload as T };

  return { ok: false, status: response.status, ...describe(payload, response.status) };
}

/** Pulls a readable message, and a machine code when there is one, out of an error body. */
function describe(payload: unknown, status: number): { message: string; code?: string } {
  const record = payload && typeof payload === 'object' ? (payload as Record<string, unknown>) : {};
  const code = typeof record.code === 'string' ? record.code : undefined;

  for (const key of ['detail', 'error', 'message', 'title']) {
    const value = record[key];
    if (typeof value === 'string' && value.trim() !== '') return { message: sentence(value), code };
  }

  return { message: `The API answered ${status}.`, code };
}

function sentence(text: string): string {
  const trimmed = text.trim();
  const capitalised = trimmed.charAt(0).toUpperCase() + trimmed.slice(1);

  return /[.!?]$/.test(capitalised) ? capitalised : `${capitalised}.`;
}

export interface Probe {
  up: boolean;
  status: number;
  ms: number;
}

/** Times one unauthenticated probe from this server. The number shown is the one measured. */
export async function probe(path: string): Promise<Probe> {
  const started = performance.now();

  try {
    const response = await fetch(`${API_URL}${path}`, { cache: 'no-store', signal: AbortSignal.timeout(3_000) });
    return { up: response.ok, status: response.status, ms: Math.round(performance.now() - started) };
  } catch {
    return { up: false, status: 0, ms: Math.round(performance.now() - started) };
  }
}
