'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

/** Re-reads the page from the server on an interval, while there is something to wait for. */
export function AutoRefresh({ everyMs }: { everyMs: number }) {
  const router = useRouter();

  useEffect(() => {
    const timer = setInterval(() => router.refresh(), everyMs);
    return () => clearInterval(timer);
  }, [router, everyMs]);

  return (
    <p className="muted block-foot" role="status">
      Checking again every {everyMs / 1000} seconds.
    </p>
  );
}
