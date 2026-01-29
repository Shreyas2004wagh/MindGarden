import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft } from 'lucide-react';

export default function JournalDetailLoading() {
    return (
        <div className="mx-auto max-w-3xl px-8 py-12">
            <div className="mb-12">
                <div className="inline-flex items-center gap-2 text-sm text-muted-foreground">
                    <ArrowLeft className="h-4 w-4" />
                    Back to journals
                </div>
            </div>

            <article className="space-y-8">
                <header className="space-y-4 border-b border-border pb-8">
                    <Skeleton className="h-5 w-48" />
                    <Skeleton className="h-9 w-3/4" />
                </header>

                <div className="space-y-4">
                    <Skeleton className="h-5 w-full" />
                    <Skeleton className="h-5 w-full" />
                    <Skeleton className="h-5 w-5/6" />
                    <Skeleton className="h-5 w-full" />
                    <Skeleton className="h-5 w-4/5" />
                    <Skeleton className="h-5 w-full" />
                    <Skeleton className="h-5 w-3/4" />
                </div>
            </article>
        </div>
    );
}
