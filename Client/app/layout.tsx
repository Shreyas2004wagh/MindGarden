import './globals.css';
import type { Metadata } from 'next';
import { Inter, Crimson_Pro } from 'next/font/google';
import { Toaster } from '@/components/ui/sonner';
import { ProgressBar } from '@/components/progress-bar';

// Warm modern sans-serif for body text and UI
const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
});

// Artistic editorial serif for titles and headings
const crimsonPro = Crimson_Pro({
  subsets: ['latin'],
  variable: '--font-serif',
  display: 'swap',
  weight: ['400', '600'],
});

export const metadata: Metadata = {
  title: 'Mind Garden',
  description: 'A privacy-first, AI-powered journaling space',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={`${inter.variable} ${crimsonPro.variable} ${inter.className}`}>
        <ProgressBar />
        {children}
        <Toaster />
      </body>
    </html>
  );
}
