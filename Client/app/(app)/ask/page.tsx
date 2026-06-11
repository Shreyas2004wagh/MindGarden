'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Skeleton } from '@/components/ui/skeleton';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';

export default function AskPage() {
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState('');
  const [loading, setLoading] = useState(false);

  const handleAsk = async () => {
    if (!question.trim()) {
      toast.error('Please enter a question');
      return;
    }

    setLoading(true);
    try {
      const response = await fetch('/api/ask', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ question: question.trim() }),
      });

      if (!response.ok) {
        throw new Error('Failed to get answer');
      }

      const data = await response.json();
      setAnswer(data.answer);
    } catch (error) {
      console.error('Error asking question:', error);
      toast.error('Failed to get an answer. Please try again.');
      setAnswer('');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl px-8 py-12">
      <div className="mb-12">
        <h1 className="text-3xl tracking-tight text-foreground">Ask AI</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Ask reflective questions about your journal entries
        </p>
      </div>

      <div className="space-y-8">
        <div className="space-y-4">
          <Textarea
            placeholder="What patterns do I notice in my recent entries?"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            className="min-h-[120px] resize-none text-base leading-relaxed"
            disabled={loading}
          />
          <Button onClick={handleAsk} disabled={!question.trim() || loading}>
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Thinking...
              </>
            ) : (
              'Ask'
            )}
          </Button>
        </div>

        {loading && (
          <div className="space-y-4 border-t border-border pt-8">
            <Skeleton className="h-4 w-64" />
            <div className="space-y-3">
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-5/6" />
              <Skeleton className="h-5 w-full" />
              <Skeleton className="h-5 w-4/5" />
            </div>
          </div>
        )}

        {answer && !loading && (
          <div className="space-y-4 border-t border-border pt-8">
            <p className="text-xs text-muted-foreground">
              Answer generated from your past journal entries
            </p>
            <div className="prose prose-invert prose-lg max-w-none">
              <p className="whitespace-pre-wrap leading-relaxed text-foreground">
                {answer}
              </p>
            </div>
          </div>
        )}

        {!answer && !loading && (
          <div className="rounded-lg border border-dashed border-border py-12 text-center">
            <p className="text-sm text-muted-foreground">
              Your answer will appear here
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
