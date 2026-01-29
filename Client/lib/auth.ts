import { createClient as createServerClient } from '@/lib/supabase/server';
import { createClient as createBrowserClient } from '@/lib/supabase/client';
import { redirect } from 'next/navigation';

/**
 * Get the current authenticated user (server-side)
 * Redirects to login if not authenticated
 */
export async function getAuthenticatedUser() {
  const supabase = await createServerClient();
  const {
    data: { user },
  } = await supabase.auth.getUser();

  if (!user) {
    redirect('/login');
  }

  return user;
}

/**
 * Get the current authenticated user (client-side)
 * Returns null if not authenticated
 */
export async function getCurrentUser() {
  const supabase = createBrowserClient();
  const {
    data: { user },
  } = await supabase.auth.getUser();

  return user;
}
