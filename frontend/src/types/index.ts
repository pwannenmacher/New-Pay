// User types
export interface OAuthConnection {
  id: number;
  user_id: number;
  provider: string;
  provider_id: string;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: number;
  email: string;
  first_name: string;
  last_name: string;
  email_verified: boolean;
  email_verified_at?: string;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
  oauth_provider?: string;
  roles?: Role[];
  oauth_connections?: OAuthConnection[];
  has_local_password?: boolean;
}

export interface UserWithRoles extends User {
  roles: Role[];
}

// Role types
export interface Role {
  id: number;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

// Permission types
export interface Permission {
  id: number;
  name: string;
  resource: string;
  action: string;
  description: string;
  created_at: string;
  updated_at: string;
}

// Audit log types
export interface AuditLog {
  id: number;
  user_id?: number;
  user_email?: string;
  action: string;
  resource: string;
  details?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

// Auth types
export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

export interface PasswordResetRequest {
  email: string;
}

export interface PasswordResetConfirm {
  token: string;
  new_password: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

// Profile update types
export interface ProfileUpdateRequest {
  first_name: string;
  last_name: string;
}

// Admin types
export interface AssignRoleRequest {
  user_id: number;
  role_id: number;
}

export interface RemoveRoleRequest {
  user_id: number;
  role_id: number;
}

// Session types
export interface Session {
  session_id: string;
  created_at: string;
  last_activity_at: string;
  ip_address: string;
  user_agent: string;
  expires_at: string;
  user_id?: number;
  user_email?: string;
  user_name?: string;
}

// API Error types
export interface ApiError {
  error: string;
  message?: string;
  details?: Record<string, string[]>;
}

// Criteria Catalog types
export type CatalogPhase = 'draft' | 'active' | 'archived';

export interface CriteriaCatalog {
  id: number;
  name: string;
  description?: string;
  valid_from: string;
  valid_until: string;
  phase: CatalogPhase;
  created_by?: number;
  created_at: string;
  updated_at: string;
  published_at?: string;
  archived_at?: string;
}

export interface Category {
  id: number;
  catalog_id: number;
  name: string;
  description?: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface Level {
  id: number;
  catalog_id: number;
  name: string;
  level_number: number;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface Path {
  id: number;
  category_id: number;
  name: string;
  description?: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface PathLevelDescription {
  id: number;
  path_id: number;
  level_id: number;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CatalogChange {
  id: number;
  catalog_id: number;
  entity_type: 'catalog' | 'category' | 'path' | 'level' | 'description';
  entity_id: number;
  field_name: string;
  old_value?: string;
  new_value?: string;
  changed_by?: number;
  changed_at: string;
}

export interface PathWithDescriptions extends Path {
  descriptions?: PathLevelDescription[];
}

export interface CategoryWithPaths extends Category {
  paths?: PathWithDescriptions[];
}

export interface CatalogWithDetails extends CriteriaCatalog {
  categories?: CategoryWithPaths[];
  levels?: Level[];
}
