import { NextResponse, type NextRequest } from 'next/server';
import { SESSION_COOKIE } from '@/lib/session';

/**
 * Ends the session and returns to sign-in.
 *
 * `reason=revoked` is how the console layout says the API refused a token it had accepted
 * before: the membership behind it was suspended or removed. Clearing the cookie has to
 * happen here, in a route handler, because a page that is rendering cannot change cookies.
 */
function signOut(request: NextRequest) {
  const target = new URL('/login', request.url);
  if (request.nextUrl.searchParams.get('reason') === 'revoked') {
    target.searchParams.set('reason', 'revoked');
  }

  const response = NextResponse.redirect(target, 303);
  response.cookies.delete(SESSION_COOKIE);

  return response;
}

export const GET = signOut;
export const POST = signOut;
