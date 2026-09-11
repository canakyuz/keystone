'use client';

/** Switches between the light and dark sheets and remembers the choice in this browser. */
export function ThemeToggle() {
  function toggle() {
    const root = document.documentElement;
    const current = root.dataset.theme ?? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    const next = current === 'dark' ? 'light' : 'dark';

    root.dataset.theme = next;
    try {
      localStorage.setItem('ks-theme', next);
    } catch {
      // Storage can be unavailable (private windows, blocked site data). The switch still
      // works for this page; it just is not remembered.
    }
  }

  return (
    <button type="button" className="chrome-link" onClick={toggle}>
      Light or dark
    </button>
  );
}
