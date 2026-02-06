'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { YearGroup } from '@/lib/time-grouping';

interface TimeNavigationProps {
    yearGroups: YearGroup[];
}

export function TimeNavigation({ yearGroups }: TimeNavigationProps) {
    const pathname = usePathname();
    const [expandedYears, setExpandedYears] = useState<Set<number>>(
        new Set(yearGroups.length > 0 ? [yearGroups[0].year] : [])
    );

    const toggleYear = (year: number) => {
        setExpandedYears((prev) => {
            const next = new Set(prev);
            if (next.has(year)) {
                next.delete(year);
            } else {
                next.add(year);
            }
            return next;
        });
    };

    if (yearGroups.length === 0) {
        return (
            <div className="px-4 py-6 text-sm text-muted-foreground/60">
                No entries yet
            </div>
        );
    }

    return (
        <div className="space-y-1 px-4 py-6">
            {yearGroups.map((yearGroup) => {
                const isExpanded = expandedYears.has(yearGroup.year);

                return (
                    <div key={yearGroup.year}>
                        {/* Year header */}
                        <button
                            onClick={() => toggleYear(yearGroup.year)}
                            className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm font-medium text-foreground/80 transition-colors hover:bg-accent/20"
                        >
                            <span>{yearGroup.year}</span>
                            <div className="flex items-center gap-2">
                                <span className="text-xs text-muted-foreground/60">
                                    {yearGroup.totalCount}
                                </span>
                                {isExpanded ? (
                                    <ChevronDown className="h-3.5 w-3.5" />
                                ) : (
                                    <ChevronRight className="h-3.5 w-3.5" />
                                )}
                            </div>
                        </button>

                        {/* Month list */}
                        {isExpanded && (
                            <div className="ml-3 mt-1 space-y-0.5 border-l border-border/30 pl-3">
                                {yearGroup.months.map((monthGroup) => (
                                    <Link
                                        key={`${yearGroup.year}-${monthGroup.month}`}
                                        href={`/journals?year=${yearGroup.year}&month=${monthGroup.month}`}
                                        className={cn(
                                            'flex items-center justify-between rounded-md px-3 py-1.5 text-sm transition-colors',
                                            pathname === '/journals' // TODO: Add active state based on query params
                                                ? 'text-muted-foreground/70 hover:bg-accent/10 hover:text-foreground/90'
                                                : 'text-muted-foreground/70 hover:bg-accent/10 hover:text-foreground/90'
                                        )}
                                    >
                                        <span>{monthGroup.monthName}</span>
                                        <span className="text-xs text-muted-foreground/50">
                                            {monthGroup.count}
                                        </span>
                                    </Link>
                                ))}
                            </div>
                        )}
                    </div>
                );
            })}
        </div>
    );
}
