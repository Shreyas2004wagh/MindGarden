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
  params: { id: string };
}) {
  const { id } = params;
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
    <div className="mx-auto max-w-[900px] px-6 py-12">
      {/* Back button - inline, subtle */}
      <div className="mb-12">
        <Link
          href="/journals"
          className="group inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5 transition-transform group-hover:-translate-x-0.5" />
          <span>Back to journals</span>
        </Link>
      </div>

      {/* Article content - calm, spacious */}
      <article className="space-y-8">
        {/* Metadata - subtle */}
        <time className="block text-sm text-muted-foreground/60">
          {formatJournalDate(journal.created_at)}
        </time>

        {/* Title - large, elegant */}
        {journal.title && (
          <h1 className="text-4xl font-semibold leading-tight tracking-tight text-foreground">
            {journal.title}
          </h1>
        )}

        {/* Content - generous spacing, calm */}
        <div className="prose prose-invert max-w-none">
          <p className="whitespace-pre-wrap text-lg leading-[1.8] text-foreground/95">
            {journal.content}
          </p>
        </div>
      </article>
    </div>
  );
}
