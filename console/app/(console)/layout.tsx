import { Chrome } from '@/components/Chrome';
import { current } from '@/lib/current';

export default async function ConsoleLayout({ children }: { children: React.ReactNode }) {
  const { user, tenant } = await current();

  return (
    <>
      <Chrome user={user} tenant={tenant} />
      <main id="main">{children}</main>
    </>
  );
}
