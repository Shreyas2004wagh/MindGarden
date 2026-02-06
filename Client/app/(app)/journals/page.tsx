import Link from 'next/link';
import { createClient } from '@/lib/supabase/server';
import type { Journal } from '@/lib/supabase/types';
import { Button } from '@/components/ui/button';
import { PenLine } from 'lucide-react';
import { getAuthenticatedUser } from '@/lib/auth';
import { JournalFeed } from '@/components/journal-feed';

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
      <div className="mb-16 flex items-center justify-between">
        <div>
          <h1 className="text-5xl tracking-tight text-foreground leading-tight">Journals</h1>
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
        <JournalFeed journals={journals} />
      )}
    </div>
  );
}
