import { createServerClient } from '@supabase/ssr';
import { NextResponse, type NextRequest } from 'next/server';

export async function proxy(request: NextRequest) {
  let response = NextResponse.next({
    request: {
      headers: request.headers,
    },
  });

  const isAuthPage = request.nextUrl.pathname.startsWith('/login') ||
                     request.nextUrl.pathname.startsWith('/register');
  const isVerifyPage = request.nextUrl.pathname.startsWith('/verify-email');
  const isCallbackPage = request.nextUrl.pathname.startsWith('/auth/callback');
  const hasSupabaseConfig =
    Boolean(process.env.NEXT_PUBLIC_SUPABASE_URL) &&
    Boolean(process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY);

  if (!hasSupabaseConfig) {
    if (isAuthPage || isVerifyPage || isCallbackPage) {
      return response;
    }
    return NextResponse.redirect(new URL('/login', request.url));
  }

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() {
          return request.cookies.getAll();
        },
        setAll(cookiesToSet) {
          cookiesToSet.forEach(({ name, value }) => request.cookies.set(name, value));
          response = NextResponse.next({
            request,
          });
          cookiesToSet.forEach(({ name, value, options }) =>
            response.cookies.set(name, value, options)
          );
        },
      },
    }
  );

  const {
    data: { user },
  } = await supabase.auth.getUser();

  if (!user && !isAuthPage && !isVerifyPage && !isCallbackPage) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  // Redirect authenticated users away from auth pages
  if (user && isAuthPage) {
    // Check if email is verified
    if (!user.email_confirmed_at) {
      return NextResponse.redirect(new URL(`/verify-email?email=${encodeURIComponent(user.email || '')}`, request.url));
    }
    return NextResponse.redirect(new URL('/journals', request.url));
  }

  // Redirect authenticated but unverified users to verify-email (except if already there)
  if (user && !user.email_confirmed_at && !isVerifyPage && !isAuthPage) {
    return NextResponse.redirect(new URL(`/verify-email?email=${encodeURIComponent(user.email || '')}`, request.url));
  }

  // Redirect verified users away from verify page
  if (user && user.email_confirmed_at && isVerifyPage) {
    return NextResponse.redirect(new URL('/journals', request.url));
  }

  // Redirect to journals from root if authenticated and verified
  if (user && user.email_confirmed_at && request.nextUrl.pathname === '/') {
    return NextResponse.redirect(new URL('/journals', request.url));
  }

  return response;
}

export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
};
