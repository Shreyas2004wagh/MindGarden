import './globals.css';
import type { Metadata } from 'next';
import { Suspense } from 'react';
import { Toaster } from '@/components/ui/sonner';
import { ProgressBar } from '@/components/progress-bar';


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
      <body>
        <Suspense fallback={null}>
          <ProgressBar />
        </Suspense>
        {children}
        <Toaster />
      </body>
    </html>
  );
}
