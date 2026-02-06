import type { Journal } from '@/lib/supabase/types';

export interface MonthGroup {
  year: number;
  month: number;
  monthName: string;
  count: number;
  journals: Journal[];
}

export interface YearGroup {
  year: number;
  months: MonthGroup[];
  totalCount: number;
}

/**
 * Group journals by year and month
 * @param journals - Array of journal entries
 * @returns Array of year groups with nested month groups
 */
export function groupJournalsByYearMonth(journals: Journal[]): YearGroup[] {
  const yearMap = new Map<number, Map<number, Journal[]>>();

  // Group journals by year and month
  journals.forEach((journal) => {
    const date = new Date(journal.created_at);
    const year = date.getFullYear();
    const month = date.getMonth(); // 0-11

    if (!yearMap.has(year)) {
      yearMap.set(year, new Map());
    }

    const monthMap = yearMap.get(year)!;
    if (!monthMap.has(month)) {
      monthMap.set(month, []);
    }

    monthMap.get(month)!.push(journal);
  });

  // Convert to array format
  const yearGroups: YearGroup[] = [];

  yearMap.forEach((monthMap, year) => {
    const months: MonthGroup[] = [];
    let totalCount = 0;

    monthMap.forEach((journalList, month) => {
      const date = new Date(year, month);
      months.push({
        year,
        month,
        monthName: date.toLocaleString('default', { month: 'long' }),
        count: journalList.length,
        journals: journalList,
      });
      totalCount += journalList.length;
    });

    // Sort months in descending order (newest first)
    months.sort((a, b) => b.month - a.month);

    yearGroups.push({
      year,
      months,
      totalCount,
    });
  });

  // Sort years in descending order (newest first)
  yearGroups.sort((a, b) => b.year - a.year);

  return yearGroups;
}
