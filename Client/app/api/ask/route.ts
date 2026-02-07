import { NextRequest, NextResponse } from 'next/server';
import { createClient } from '@/lib/supabase/server';

const LOCAL_BACKEND_DEFAULTS = [
  'http://localhost:8082',
  'http://localhost:8080',
  'http://localhost:8081',
];

function normalizeBaseUrl(rawBaseUrl: string): string | null {
  const trimmed = rawBaseUrl.trim();
  if (!trimmed) {
    return null;
  }

  try {
    const parsed = new URL(trimmed);
    const path = parsed.pathname.replace(/\/+$/, '');
    return `${parsed.origin}${path}`;
  } catch {
    return null;
  }
}

function buildBackendBaseCandidates(): string[] {
  const configured = [process.env.API_URL || '', process.env.NEXT_PUBLIC_API_URL || ''];

  const normalizedConfigured = configured
    .map(normalizeBaseUrl)
    .filter((url): url is string => Boolean(url));

  const normalizedFallbacks = LOCAL_BACKEND_DEFAULTS
    .map(normalizeBaseUrl)
    .filter((url): url is string => Boolean(url));

  return Array.from(new Set([...normalizedConfigured, ...normalizedFallbacks]));
}

async function fetchWithTimeout(
  endpoint: string,
  init: RequestInit,
  timeoutMs: number
): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    return await fetch(endpoint, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timeoutId);
  }
}

function buildAskEndpointCandidates(rawBaseUrl: string): string[] {
  const normalizedBaseUrl = normalizeBaseUrl(rawBaseUrl);
  if (!normalizedBaseUrl) {
    return [];
  }

  const parsed = new URL(normalizedBaseUrl);
  const path = parsed.pathname.replace(/\/+$/, '');
  const baseWithPath = `${parsed.origin}${path}`;

  const candidates: string[] = [];

  if (path.endsWith('/api')) {
    // Handles configs like http://localhost:8080/api
    candidates.push(`${baseWithPath}/ask`);
    const rootPath = path.slice(0, -4);
    candidates.push(`${parsed.origin}${rootPath}/ask`);
  } else {
    candidates.push(`${baseWithPath}/ask`);
    candidates.push(`${baseWithPath}/api/ask`);
  }

  return Array.from(new Set(candidates));
}

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

    // Forward the request to the Go backend (supports /ask and /api/ask backends)
    const backendBaseUrls = buildBackendBaseCandidates();
    const askEndpoints = backendBaseUrls.flatMap(buildAskEndpointCandidates);

    let response: Response | null = null;
    let resolvedEndpoint = '';
    const attemptedEndpoints: string[] = [];
    const networkErrors: string[] = [];

    for (const endpoint of askEndpoints) {
      attemptedEndpoints.push(endpoint);

      try {
        const attempted = await fetchWithTimeout(
          endpoint,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${session.access_token}`,
            },
            body: JSON.stringify({
              question: question.trim(),
              user_id: user.id,
            }),
          },
          3000
        );

        if (attempted.status === 404) {
          continue;
        }

        response = attempted;
        resolvedEndpoint = endpoint;
        break;
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        networkErrors.push(`${endpoint}: ${message}`);
      }
    }

    if (!response) {
      const attempted = attemptedEndpoints.length > 0 ? attemptedEndpoints.join(', ') : 'none';
      const networkDetail =
        networkErrors.length > 0 ? ` Network errors: ${networkErrors.join(' | ')}` : '';

      return NextResponse.json(
        { error: `Ask backend route not reachable. Tried: ${attempted}.${networkDetail}` },
        { status: 502 }
      );
    }

    if (!response.ok) {
      const errorText = await response.text();
      console.error(`Backend error from ${resolvedEndpoint}:`, errorText);
      return NextResponse.json(
        { error: `Ask endpoint failed (${response.status}): ${errorText || 'Unknown error'}` },
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
