import Link from 'next/link';
import { createClient } from '@/lib/supabase/server';
import type { Journal } from '@/lib/supabase/types';
import { Button } from '@/components/ui/button';
import { PenLine } from 'lucide-react';
import { getAuthenticatedUser } from '@/lib/auth';
import { formatJournalDate, getJournalPreview, getJournalDisplayTitle } from '@/lib/journal-utils';

export default async function JournalsPage() {
  const supabase = await createClient();
  const user = await getAuthenticatedUser();

  const { data } = await supabase
    .from('journals')
    .select('*')
    .eq('user_id', user.id)
    .order('created_at', { ascending: false });

  const journals = (data as Journal[]) || [];

  return (
    <div className="mx-auto max-w-4xl px-8 py-12">
      <div className="mb-12 flex items-center justify-between">
        <div>
          <h1 className="text-3xl tracking-tight text-foreground">Journals</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Your personal thinking space
          </p>
        </div>
        <Link href="/journals/new">
          <Button className="gap-2">
            <PenLine className="h-4 w-4" />
            New Entry
          </Button>
        </Link>
      </div>

      {journals.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24">
          <p className="mb-6 text-center text-muted-foreground">
            No journal entries yet
          </p>
          <Link href="/journals/new">
            <Button className="gap-2">
              <PenLine className="h-4 w-4" />
              Write your first entry
            </Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-8">
          {journals.map((journal) => (
            <Link
              key={journal.id}
              href={`/journals/${journal.id}`}
              className="group block"
            >
              <article className="space-y-2 rounded-lg border border-transparent px-4 py-5 transition-colors hover:border-border hover:bg-card/50">
                <time className="text-xs text-muted-foreground">
                  {formatJournalDate(journal.created_at)}
                </time>
                <h2 className="text-lg text-foreground transition-colors group-hover:text-primary">
                  {getJournalDisplayTitle(journal)}
                </h2>
                <p className="leading-relaxed text-muted-foreground">
                  {getJournalPreview(journal.content)}
                </p>
              </article>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
