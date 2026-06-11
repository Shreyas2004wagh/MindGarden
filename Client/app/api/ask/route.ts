import { NextRequest, NextResponse } from 'next/server';
import { createClient } from '@/lib/supabase/server';

export async function POST(request: NextRequest) {
  try {
    const supabase = await createClient();
    const {
      data: { user },
    } = await supabase.auth.getUser();

    if (!user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const { question } = await request.json();

    if (!question || typeof question !== 'string' || !question.trim()) {
      return NextResponse.json({ error: 'Question is required' }, { status: 400 });
    }

    // Get the session token for backend authentication
    const {
      data: { session },
    } = await supabase.auth.getSession();

    if (!session) {
      return NextResponse.json({ error: 'No session found' }, { status: 401 });
    }

    // Forward the request to the Go backend
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    const response = await fetch(`${backendUrl}/ask`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${session.access_token}`,
      },
      body: JSON.stringify({
        question: question.trim(),
        user_id: user.id,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error('Backend error:', errorText);
      return NextResponse.json(
        { error: 'Failed to get answer from AI' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error in ask API:', error);
    return NextResponse.json(
      { error: 'An error occurred processing your question' },
      { status: 500 }
    );
  }
}
