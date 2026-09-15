'use client';

/**
 * Switches between the light and dark sheets and remembers the choice in a cookie, which the
 * root layout reads so the next page is rendered in the chosen sheet from the start.
 */
export function ThemeToggle() {
  function toggle() {
    const root = document.documentElement;
    const current = root.dataset.theme ?? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    const next = current === 'dark' ? 'light' : 'dark';

    root.dataset.theme = next;
    // A year, the whole site, and never sent to another site. It holds a display preference
    // and nothing else, so it needs no protection beyond that.
    document.cookie = `ks-theme=${next}; path=/; max-age=31536000; samesite=lax`;
  }

  return (
    <button type="button" className="chrome-link" onClick={toggle}>
      Light or dark
    </button>
  );
}
