import type { Role } from './types';

export const ROLES: Role[] = ['owner', 'admin', 'editor', 'viewer'];

/**
 * Whether to offer member management. A convenience for the interface only: the API
 * checks the role on every request whatever this says.
 */
export function canManageMembers(role: Role): boolean {
  return role === 'owner' || role === 'admin';
}
