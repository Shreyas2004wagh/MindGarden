'use client';

import { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { createClient } from '@/lib/supabase/client';
import { ArrowLeft, Check } from 'lucide-react';
import Link from 'next/link';
import { toast } from 'sonner';

export default function NewJournalPage() {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [saving, setSaving] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto-resize textarea as content grows
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = textareaRef.current.scrollHeight + 'px';
    }
  }, [content]);

  const handleSave = async () => {
    if (!content.trim()) {
      toast.error('Please write something before saving');
      return;
    }

    setSaving(true);
    try {
      const supabase = createClient();
      const {
        data: { session },
      } = await supabase.auth.getSession();

      if (!session) {
        toast.error('You must be logged in to save a journal');
        router.push('/login');
        return;
      }

      // Call Go backend API instead of direct Supabase insert
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const response = await fetch(`${apiUrl}/journals`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.access_token}`,
        },
        body: JSON.stringify({
          title: title.trim() || null,
          content: content.trim(),
        }),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || 'Failed to save journal entry');
      }

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
    <div className="mx-auto max-w-[900px] px-6 py-12">
      {/* Back and Save buttons - inline, subtle */}
      <div className="mb-12 flex items-center justify-between">
        <Link
          href="/journals"
          className="group flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5 transition-transform group-hover:-translate-x-0.5" />
          <span>Back</span>
        </Link>

        <button
          onClick={handleSave}
          disabled={!content.trim() || saving}
          className="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-all disabled:opacity-40 disabled:cursor-not-allowed enabled:hover:bg-accent/50 enabled:text-foreground text-muted-foreground"
        >
          {saving ? (
            <>
              <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
              <span>Saving...</span>
            </>
          ) : (
            <>
              <Check className="h-3.5 w-3.5" />
              <span>Save</span>
            </>
          )}
        </button>
      </div>

      {/* Editor - calm, spacious */}
      <div className="space-y-8 pb-24">
        {/* Title input - large, elegant */}
        <input
          type="text"
          placeholder="Untitled"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="w-full border-0 bg-transparent px-0 text-4xl font-semibold leading-tight tracking-tight text-foreground placeholder:text-muted-foreground/20 focus:outline-none focus:ring-0"
          style={{ caretColor: 'hsl(var(--foreground))' }}
        />

        {/* Body textarea - auto-expanding, no internal scroll */}
        <textarea
          ref={textareaRef}
          placeholder="Start writing..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          className="min-h-[60vh] w-full resize-none overflow-hidden border-0 bg-transparent px-0 text-lg leading-[1.8] text-foreground placeholder:text-muted-foreground/20 focus:outline-none focus:ring-0"
          autoFocus
          style={{ caretColor: 'hsl(var(--foreground))' }}
        />
      </div>
    </div>
  );
}
