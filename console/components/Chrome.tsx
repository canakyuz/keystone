import Link from 'next/link';
import type { Tenant, User } from '@/lib/types';
import { NavLink } from './NavLink';
import { Tag } from './Tag';
import { ThemeToggle } from './ThemeToggle';

export function Chrome({ user, tenant }: { user: User; tenant: Tenant }) {
  return (
    <header className="chrome">
      <Link href="/" className="chrome-brand">
        Keystone
      </Link>
      <span className="chrome-tenant mono">{tenant.slug}</span>
      <nav className="chrome-nav" aria-label="Console">
        <NavLink href="/">Tenant</NavLink>
        <NavLink href="/members">Members</NavLink>
        <NavLink href="/provision">New tenant</NavLink>
      </nav>
      <div className="chrome-meta">
        <span className="chrome-subject">{user.email}</span>
        <Tag tone="accent">{user.role}</Tag>
        <ThemeToggle />
        <form action="/logout" method="post">
          <button className="chrome-link" type="submit">
            Sign out
          </button>
        </form>
      </div>
    </header>
  );
}
