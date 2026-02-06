import { Sidebar } from '@/components/layout/sidebar';

export default function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="relative z-10 flex h-screen overflow-hidden bg-transparent">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        {children}
      </main>
      {/* Right contextual panel - reserved for future calendar */}
      <aside className="w-80 border-l border-white/[0.08] bg-black/30 backdrop-blur-xl">
        {/* Placeholder for journal calendar */}
      </aside>
    </div>
  );
}
