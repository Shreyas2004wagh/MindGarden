import { Skeleton } from '@/components/ui/skeleton';

export default function JournalsLoading() {
    return (
        <div className="mx-auto max-w-4xl px-8 py-12">
            <div className="mb-12 flex items-center justify-between">
                <div>
                    <Skeleton className="h-9 w-32" />
                    <Skeleton className="mt-2 h-5 w-48" />
                </div>
                <Skeleton className="h-10 w-32" />
            </div>

            <div className="space-y-8">
                {[1, 2, 3, 4, 5].map((i) => (
                    <div key={i} className="space-y-2 rounded-lg border border-border px-4 py-5">
                        <Skeleton className="h-4 w-32" />
                        <Skeleton className="h-6 w-3/4" />
                        <Skeleton className="h-5 w-full" />
                        <Skeleton className="h-5 w-5/6" />
                    </div>
                ))}
            </div>
        </div>
    );
}
