import Link from 'next/link';
import { notFound } from 'next/navigation';
import { createClient } from '@/lib/supabase/server';
import type { Journal } from '@/lib/supabase/types';
import { ArrowLeft } from 'lucide-react';
import { getAuthenticatedUser } from '@/lib/auth';
import { formatJournalDate } from '@/lib/journal-utils';

export default async function JournalDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const supabase = await createClient();
  const user = await getAuthenticatedUser();

  const { data } = await supabase
    .from('journals')
    .select('*')
    .eq('id', id)
    .eq('user_id', user.id)
    .maybeSingle();

  const journal = data as Journal | null;

  if (!journal) {
    notFound();
  }

  return (
    <div className="mx-auto max-w-3xl px-8 py-12">
      <div className="mb-12">
        <Link
          href="/journals"
          className="inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to journals
        </Link>
      </div>

      <article className="space-y-8">
        <header className="space-y-4 border-b border-border pb-8">
          <time className="text-sm text-muted-foreground">
            {formatJournalDate(journal.created_at)}
          </time>
          {journal.title && (
            <h1 className="text-3xl tracking-tight text-foreground">
              {journal.title}
            </h1>
          )}
        </header>

        <div className="prose prose-invert prose-lg max-w-none">
          <p className="whitespace-pre-wrap leading-relaxed text-foreground">
            {journal.content}
          </p>
        </div>
      </article>
    </div>
  );
}
