export type Tone = 'live' | 'wip' | 'refused' | 'accent' | 'plain';

const TONES: Record<string, Tone> = {
  active: 'live',
  succeeded: 'live',
  trial: 'wip',
  pending: 'wip',
  running: 'wip',
  provisioning: 'wip',
  suspended: 'refused',
  failed: 'refused',
};

export function toneFor(status: string): Tone {
  return TONES[status] ?? 'plain';
}

const dateTime = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: 'UTC',
});

const time = new Intl.DateTimeFormat('en-GB', {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  timeZone: 'UTC',
});

export function when(iso?: string | null): string {
  if (!iso) return 'Never';
  const date = new Date(iso);

  return Number.isNaN(date.getTime()) ? 'Unknown' : `${dateTime.format(date)} UTC`;
}

export function clock(date: Date): string {
  return `${time.format(date)} UTC`;
}

export function took(fromIso: string, toIso?: string): string {
  const ms = (toIso ? new Date(toIso).getTime() : Date.now()) - new Date(fromIso).getTime();
  if (ms < 1_000) return `${Math.max(ms, 0)} ms`;
  if (ms < 90_000) return `${(ms / 1_000).toFixed(1)} s`;

  return `${Math.round(ms / 60_000)} min`;
}

export function bytes(n: number): string {
  if (n < 0) return 'Unlimited';
  if (n >= 1024 ** 3) return `${Math.round(n / 1024 ** 3)} GB`;
  if (n >= 1024 ** 2) return `${Math.round(n / 1024 ** 2)} MB`;

  return `${Math.round(n / 1024)} KB`;
}

export function limit(n: number): string {
  return n < 0 ? 'Unlimited' : String(n);
}

export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

export const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
