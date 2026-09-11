import 'server-only';

/**
 * Where the Keystone API answers. Read on the server only: the browser never talks to the
 * API directly, so there is no API address to leak and no cross-origin policy to open.
 */
export const API_URL = (process.env.KEYSTONE_API_URL ?? 'http://127.0.0.1:8099').replace(/\/+$/, '');
