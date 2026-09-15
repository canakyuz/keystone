import type { Metadata } from 'next';
import { Anton, Inter, Inter_Tight, JetBrains_Mono } from 'next/font/google';
import { cookies } from 'next/headers';
import './globals.css';

const anton = Anton({ subsets: ['latin'], weight: '400', variable: '--font-anton' });
const inter = Inter({ subsets: ['latin'], variable: '--font-inter' });
const interTight = Inter_Tight({ subsets: ['latin'], variable: '--font-inter-tight' });
const mono = JetBrains_Mono({ subsets: ['latin'], variable: '--font-jetbrains-mono' });

export const metadata: Metadata = {
  title: { default: 'Keystone', template: '%s | Keystone' },
  description: 'The console for a Keystone control plane: the tenant, its members, and the work provisioning does.',
};

// The reader's theme choice is a cookie the toggle writes, read here so the server renders the
// right sheet from the first byte. It used to live in localStorage and be applied by an inline
// script, which React 19 warns about and which ran only after the page had started to paint.
// With no choice made, the stylesheet follows the system setting.
async function chosenTheme(): Promise<'light' | 'dark' | undefined> {
  const value = (await cookies()).get('ks-theme')?.value;
  return value === 'light' || value === 'dark' ? value : undefined;
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      data-theme={await chosenTheme()}
      className={`${anton.variable} ${inter.variable} ${interTight.variable} ${mono.variable}`}
    >
      <body>
        <a className="skip-link" href="#main">
          Skip to content
        </a>
        {children}
      </body>
    </html>
  );
}
