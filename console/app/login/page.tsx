import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { ThemeToggle } from '@/components/ThemeToggle';
import { readToken } from '@/lib/session';
import { LoginForm } from './LoginForm';

export const metadata: Metadata = { title: 'Sign in' };

const NOTICES: Record<string, string> = {
  expired: 'Your session expired. Sign in again.',
  revoked: 'Your session ended because the API no longer accepts your membership in that tenant.',
};

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ reason?: string }> }) {
  if (await readToken()) redirect('/');

  const { reason } = await searchParams;

  return (
    <div className="signin">
      <header className="chrome">
        <span className="chrome-brand">Keystone</span>
        <div className="chrome-meta">
          <ThemeToggle />
        </div>
      </header>

      <main className="signin-body">
        <div>
          <h1 className="masthead-name">Tenants kept apart</h1>
          <p className="masthead-line">
            The console for a Keystone control plane. It reads through the same API a client would, and meets the same
            checks: your token, then your membership in the tenant, then your role.
          </p>
        </div>

        <section aria-labelledby="signin-title">
          <h2 id="signin-title" className="block-title signin-title">
            Sign in
          </h2>
          <LoginForm notice={reason ? (NOTICES[reason] ?? null) : null} />
        </section>
      </main>
    </div>
  );
}
