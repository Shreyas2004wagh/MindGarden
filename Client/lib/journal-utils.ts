import { format } from 'date-fns';
import type { Journal } from '@/lib/supabase/types';

/**
 * Format a date string to a readable format with time
 * @param dateString - ISO date string
 * @returns Formatted date like "Monday, January 30, 2026 • 10:31 PM"
 */
export function formatJournalDate(dateString: string): string {
  const date = new Date(dateString);
  const dateStr = format(date, 'EEEE, MMMM d, yyyy');
  const timeStr = format(date, 'h:mm a');
  return `${dateStr} • ${timeStr}`;
}

/**
 * Get a preview of journal content
 * @param content - Full journal content
 * @param maxLength - Maximum length of preview (default: 120)
 * @returns Truncated content with ellipsis if needed
 */
export function getJournalPreview(content: string, maxLength: number = 120): string {
  if (content.length <= maxLength) return content;
  return content.substring(0, maxLength).trim() + '...';
}

/**
 * Get display title for a journal entry
 * Falls back to first line of content if no title
 * @param journal - Journal entry
 * @returns Display title
 */
export function getJournalDisplayTitle(journal: Journal): string {
  if (journal.title) return journal.title;
  const firstLine = journal.content.split('\n')[0];
  return firstLine.length > 60 ? firstLine.substring(0, 60) + '...' : firstLine;
}
