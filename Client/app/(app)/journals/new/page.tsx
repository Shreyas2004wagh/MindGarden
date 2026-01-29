'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { createClient } from '@/lib/supabase/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { ArrowLeft } from 'lucide-react';
import Link from 'next/link';
import { toast } from 'sonner';

export default function NewJournalPage() {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    if (!content.trim()) {
      toast.error('Please write something before saving');
      return;
    }

    setSaving(true);
    try {
      const supabase = createClient();
      const {
        data: { user },
      } = await supabase.auth.getUser();

      if (!user) {
        toast.error('You must be logged in to save a journal');
        router.push('/login');
        return;
      }

      const { error } = await supabase
        .from('journals')
        .insert({
          user_id: user.id,
          title: title.trim() || null,
          content: content.trim(),
        } as any);

      if (error) throw error;

      toast.success('Journal entry saved successfully!');
      router.push('/journals');
      router.refresh();
    } catch (error: any) {
      console.error('Error saving journal:', error);
      toast.error(error.message || 'Failed to save journal entry');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl px-8 py-12">
      <div className="mb-8 flex items-center justify-between">
        <Link
          href="/journals"
          className="flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Link>
        <Button onClick={handleSave} disabled={!content.trim() || saving}>
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </div>

      <div className="space-y-6">
        <Input
          type="text"
          placeholder="Title (optional)"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="border-0 bg-transparent px-0 text-2xl placeholder:text-muted-foreground/40 focus-visible:ring-0"
        />

        <Textarea
          placeholder="Start writing..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          className="min-h-[500px] resize-none border-0 bg-transparent px-0 text-base leading-relaxed placeholder:text-muted-foreground/40 focus-visible:ring-0"
          autoFocus
        />
      </div>
    </div>
  );
}
