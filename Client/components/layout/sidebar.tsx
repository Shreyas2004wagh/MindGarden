'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { BookOpen, MessageSquare, LogOut } from 'lucide-react';
import { createClient } from '@/lib/supabase/client';
import { cn } from '@/lib/utils';
import { toast } from 'sonner';
import NProgress from 'nprogress';

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const supabase = createClient();

  const handleLogout = async () => {
    await supabase.auth.signOut();
    toast.success('Logged out successfully');
    router.push('/login');
  };

  const navigation = [
    { name: 'Journals', href: '/journals', icon: BookOpen },
    { name: 'Ask AI', href: '/ask', icon: MessageSquare },
  ];

  const handleLinkClick = () => {
    NProgress.start();
  };

  return (
    <div className="flex h-full w-64 flex-col border-r border-border/40 bg-card/30">
      {/* Logo/Brand - subtle */}
      <div className="flex h-16 items-center border-b border-border/30 px-6">
        <h1 className="text-lg font-medium tracking-tight text-foreground/80">
          Mind Garden
        </h1>
      </div>

      {/* Navigation - muted */}
      <nav className="flex-1 space-y-1 px-4 py-6">
        {navigation.map((item) => {
          const isActive = pathname === item.href || pathname?.startsWith(item.href + '/');
          return (
            <Link
              key={item.name}
              href={item.href}
              onClick={handleLinkClick}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                isActive
                  ? 'bg-accent/40 text-foreground'
                  : 'text-muted-foreground/70 hover:bg-accent/20 hover:text-foreground/90'
              )}
            >
              <item.icon className="h-4 w-4" />
              {item.name}
            </Link>
          );
        })}
      </nav>

      {/* Logout - subtle */}
      <div className="border-t border-border/30 p-4">
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground/70 transition-all hover:bg-accent/20 hover:text-foreground/90"
        >
          <LogOut className="h-4 w-4" />
          Logout
        </button>
      </div>
    </div>
  );
}
