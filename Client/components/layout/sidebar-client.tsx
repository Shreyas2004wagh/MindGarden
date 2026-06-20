'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { BookOpen, MessageSquare, LogOut } from 'lucide-react';
import { createClient } from '@/lib/supabase/client';
import { cn } from '@/lib/utils';
import { toast } from 'sonner';
import NProgress from 'nprogress';

export function SidebarClient() {
    const pathname = usePathname();
    const router = useRouter();
    const handleLogout = async () => {
        const supabase = createClient();
        await supabase.auth.signOut();
        toast.success('Logged out successfully');
        router.push('/login');
    };

    const handleLinkClick = () => {
        NProgress.start();
    };

    const isJournalsActive = pathname === '/journals' || pathname?.startsWith('/journals/');
    const isAskActive = pathname === '/ask' || pathname?.startsWith('/ask/');

    return (
        <div className="flex h-full w-64 flex-col border-r border-white/[0.08] bg-black/30 backdrop-blur-xl">
            {/* Logo/Brand - subtle */}
            <div className="flex h-16 items-center border-b border-white/[0.08] px-6">
                <h1 className="text-lg font-medium tracking-tight text-foreground/80">
                    Mind Garden
                </h1>
            </div>

            {/* Navigation */}
            <nav className="flex-1 space-y-1 px-4 py-6">
                <Link
                    href="/journals"
                    onClick={handleLinkClick}
                    className={cn(
                        'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                        isJournalsActive
                            ? 'bg-accent/40 text-foreground'
                            : 'text-muted-foreground/70 hover:bg-accent/20 hover:text-foreground/90'
                    )}
                >
                    <BookOpen className="h-4 w-4" />
                    Journals
                </Link>

                <Link
                    href="/ask"
                    onClick={handleLinkClick}
                    className={cn(
                        'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                        isAskActive
                            ? 'bg-accent/40 text-foreground'
                            : 'text-muted-foreground/70 hover:bg-accent/20 hover:text-foreground/90'
                    )}
                >
                    <MessageSquare className="h-4 w-4" />
                    Ask AI
                </Link>
            </nav>

            {/* Logout - subtle */}
            <div className="border-t border-white/[0.08] p-4">
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
