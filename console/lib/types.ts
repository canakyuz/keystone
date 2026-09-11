/** The shapes the Keystone API answers with, as far as this console reads them. */

export type Role = 'owner' | 'admin' | 'editor' | 'viewer';

export interface User {
  id: string;
  tenant_id: string;
  email: string;
  first_name: string;
  last_name: string;
  full_name: string;
  role: Role;
  status: string;
  created_at: string;
  updated_at: string;
  email_verified: boolean;
  last_login_at?: string;
}

export interface UserList {
  data: User[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  email: string;
  schema_name: string;
  status: string;
  plan: string;
  created_at: string;
  updated_at: string;
  trial_ends_at?: string;
  custom_domain?: string;
  feature_limits: {
    max_users: number;
    max_websites: number;
    max_storage: number;
    custom_domain: boolean;
    api_access: boolean;
  };
}

export type OperationStatus = 'pending' | 'running' | 'succeeded' | 'failed';

export interface Operation {
  id: string;
  tenant_id: string;
  kind: string;
  status: OperationStatus;
  error_code?: string;
  error_detail?: string;
  created_at: string;
  completed_at?: string;
}

export interface LoginResult {
  user: User;
  access_token: string;
  expires_in: number;
}

/** Most Keystone handlers wrap their answer in `data`. */
export interface Envelope<T> {
  data: T;
}
