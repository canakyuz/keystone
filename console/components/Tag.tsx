import { toneFor, type Tone } from '@/lib/format';

export function Tag({ children, tone = 'plain' }: { children: React.ReactNode; tone?: Tone }) {
  return <span className={tone === 'plain' ? 'tag' : `tag tag--${tone}`}>{children}</span>;
}

export function StatusTag({ status }: { status: string }) {
  return <Tag tone={toneFor(status)}>{status.replaceAll('_', ' ')}</Tag>;
}
