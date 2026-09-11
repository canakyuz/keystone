import 'server-only';
import { cookies } from 'next/headers';

/**
 * The session is the Keystone access token, kept in an httpOnly cookie.
 *
 * httpOnly because nothing in the browser needs to read it: every call to the API is made
 * from the server. A token that script can read is a token any injected script can send
 * somewhere else.
 */
export const SESSION_COOKIE = 'ks_session';

export async function readToken(): Promise<string | null> {
  return (await cookies()).get(SESSION_COOKIE)?.value ?? null;
}

export async function writeToken(token: string, maxAgeSeconds: number): Promise<void> {
  (await cookies()).set(SESSION_COOKIE, token, {
    httpOnly: true,
    sameSite: 'lax',
    secure: process.env.NODE_ENV === 'production',
    path: '/',
    maxAge: maxAgeSeconds,
  });
}
