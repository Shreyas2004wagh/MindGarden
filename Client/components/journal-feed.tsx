'use client';

import { useState, useEffect } from 'react';
import type { Journal } from '@/lib/supabase/types';
import { formatJournalDate, getJournalPreview, getJournalDisplayTitle } from '@/lib/journal-utils';

interface JournalFeedProps {
    journals: Journal[];
}

export function JournalFeed({ journals }: JournalFeedProps) {
    const [focusedId, setFocusedId] = useState<string | null>(null);

    // Get the focused journal
    const focusedJournal = focusedId ? journals.find(j => j.id === focusedId) : null;

    // Handle escape key to exit focus mode
    useEffect(() => {
        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && focusedId) {
                setFocusedId(null);
            }
        };

        window.addEventListener('keydown', handleEscape);
        return () => window.removeEventListener('keydown', handleEscape);
    }, [focusedId]);

    // Lock body scroll when in focus mode
    useEffect(() => {
        if (focusedId) {
            document.body.style.overflow = 'hidden';
        } else {
            document.body.style.overflow = '';
        }

        return () => {
            document.body.style.overflow = '';
        };
    }, [focusedId]);

    const handleEntryClick = (journalId: string) => {
        setFocusedId(journalId);
    };

    const handleBackdropClick = () => {
        setFocusedId(null);
    };

    return (
        <>
            {/* Backdrop overlay when in focus mode */}
            {focusedId && (
                <div
                    className="fixed inset-0 z-40 bg-black/80 backdrop-blur-sm transition-opacity duration-300"
                    onClick={handleBackdropClick}
                />
            )}

            <div className={`space-y-3 transition-all duration-300 ${focusedId ? 'relative z-50' : ''}`}>
                {journals.map((journal) => {
                    const isFocused = focusedId === journal.id;
                    const isDimmed = focusedId && !isFocused;

                    return (
                        <div
                            key={journal.id}
                            onClick={(e) => {
                                // If focused and clicking the wrapper (not the card), close it
                                if (isFocused && e.target === e.currentTarget) {
                                    setFocusedId(null);
                                }
                            }}
                            className={`
                group cursor-pointer transition-all duration-300
                ${isFocused ? 'fixed inset-0 z-50 flex items-center justify-center p-8 overflow-y-auto' : ''}
                ${isDimmed ? 'opacity-30 blur-sm pointer-events-none' : 'opacity-100'}
              `}
                        >
                            <article
                                onClick={() => !isFocused && handleEntryClick(journal.id)}
                                className={`
                  space-y-4 rounded-xl px-8 py-8
                  transition-all duration-300
                  ${isFocused
                                        ? 'bg-black/90 border border-white/[0.1] shadow-2xl shadow-black/20 max-w-4xl w-full my-auto relative'
                                        : 'bg-black/20 backdrop-blur-sm border border-white/[0.06] shadow-lg shadow-black/10 hover:bg-black/25 hover:border-white/[0.08] hover:shadow-xl hover:shadow-black/15 hover:-translate-y-0.5'
                                    }
                `}
                            >
                                {/* Header with date and close button aligned */}
                                {isFocused ? (
                                    <div className="flex items-center justify-between mb-4">
                                        <time className="block text-[15px] tracking-wide uppercase text-white/70">
                                            {formatJournalDate(journal.created_at)}
                                        </time>
                                        <button
                                            onClick={() => setFocusedId(null)}
                                            className="w-8 h-8 flex items-center justify-center rounded-full bg-white/5 hover:bg-white/10 border border-white/10 hover:border-white/20 transition-all duration-200 group/close"
                                            aria-label="Close"
                                        >
                                            <svg
                                                className="w-4 h-4 text-white/60 group-hover/close:text-white/90 transition-colors"
                                                fill="none"
                                                viewBox="0 0 24 24"
                                                stroke="currentColor"
                                            >
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                            </svg>
                                        </button>
                                    </div>
                                ) : (
                                    <time className="block text-[15px] tracking-wide uppercase text-white/70">
                                        {formatJournalDate(journal.created_at)}
                                    </time>
                                )}

                                {journal.title && (
                                    <h2 className="text-2xl leading-snug text-foreground">
                                        {journal.title}
                                    </h2>
                                )}

                                {!journal.title && (
                                    <h2 className="text-2xl leading-snug text-foreground">
                                        {getJournalDisplayTitle(journal)}
                                    </h2>
                                )}

                                {/* Show preview when not focused, full content when focused */}
                                {isFocused ? (
                                    <div className="prose prose-invert max-w-none">
                                        <p className="whitespace-pre-wrap text-base leading-[1.8] text-foreground/90">
                                            {journal.content}
                                        </p>
                                    </div>
                                ) : (
                                    <p className="text-[15px] leading-relaxed text-white/85">
                                        {getJournalPreview(journal.content)}
                                    </p>
                                )}
                            </article>
                        </div>
                    );
                })}
            </div>
        </>
    );
}
