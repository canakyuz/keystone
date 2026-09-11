import type { Metadata } from 'next';
import { Anton, Inter, Inter_Tight, JetBrains_Mono } from 'next/font/google';
import './globals.css';

const anton = Anton({ subsets: ['latin'], weight: '400', variable: '--font-anton' });
const inter = Inter({ subsets: ['latin'], variable: '--font-inter' });
const interTight = Inter_Tight({ subsets: ['latin'], variable: '--font-inter-tight' });
const mono = JetBrains_Mono({ subsets: ['latin'], variable: '--font-jetbrains-mono' });

export const metadata: Metadata = {
  title: { default: 'Keystone', template: '%s | Keystone' },
  description: 'The console for a Keystone control plane: the tenant, its members, and the work provisioning does.',
};

// Applied before the first paint, so a reader who chose dark does not see a flash of paper.
const themeScript = `try{var t=localStorage.getItem('ks-theme');if(t)document.documentElement.dataset.theme=t}catch(e){}`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${anton.variable} ${inter.variable} ${interTight.variable} ${mono.variable}`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body>{children}</body>
    </html>
  );
}
