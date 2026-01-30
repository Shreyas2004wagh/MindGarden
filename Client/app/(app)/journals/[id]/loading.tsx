import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft } from 'lucide-react';

export default function JournalDetailLoading() {
    return (
        <div className="mx-auto max-w-[900px] px-6 py-12">
            {/* Back button skeleton - inline, subtle */}
            <div className="mb-12">
                <div className="inline-flex items-center gap-2 text-sm text-muted-foreground">
                    <ArrowLeft className="h-3.5 w-3.5" />
                    Back to journals
                </div>
            </div>

            {/* Article content skeleton - calm, spacious */}
            <article className="space-y-8">
                {/* Timestamp skeleton - subtle */}
                <Skeleton className="h-4 w-64" />

                {/* Title skeleton - large */}
                <Skeleton className="h-12 w-3/4" />

                {/* Content skeleton - generous spacing */}
                <div className="space-y-4 pt-4">
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-11/12" />
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-5/6" />
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-4/5" />
                    <Skeleton className="h-6 w-full" />
                    <Skeleton className="h-6 w-3/4" />
                </div>
            </article>
        </div>
    );
}
