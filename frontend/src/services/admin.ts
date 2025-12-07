import apiClient from './api';
import type { UserWithRoles, Role, AuditLog, AssignRoleRequest, RemoveRoleRequest, Session } from '../types';

export interface PaginatedResponse<T> {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  users?: T[];
  logs?: AuditLog[];
}

export interface UserListParams {
  page?: number;
  limit?: number;
  search?: string;
  role_ids?: number[];
  is_active?: boolean;
  email_verified?: boolean;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface AuditLogListParams {
  page?: number;
  limit?: number;
  user_id?: number;
  action?: string;
  resource?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export const adminApi = {
  // User management
  getUser: (userId: number) =>
    apiClient.get<UserWithRoles>(`/admin/users/get?id=${userId}`),
  
  listUsers: (params: UserListParams = {}) => {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.search) queryParams.append('search', params.search);
    if (params.role_ids && params.role_ids.length > 0) {
      queryParams.append('role_ids', params.role_ids.join(','));
    }
    if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString());
    if (params.email_verified !== undefined) queryParams.append('email_verified', params.email_verified.toString());
    if (params.sort_by) queryParams.append('sort_by', params.sort_by);
    if (params.sort_order) queryParams.append('sort_order', params.sort_order);
    
    return apiClient.get<PaginatedResponse<UserWithRoles>>(`/admin/users/list?${queryParams.toString()}`);
  },
  
  assignRole: (data: AssignRoleRequest) =>
    apiClient.post<{ message: string }>('/admin/users/assign-role', data),
  
  removeRole: (data: RemoveRoleRequest) =>
    apiClient.post<{ message: string }>('/admin/users/remove-role', data),
  
  updateUserStatus: (userId: number, isActive: boolean) =>
    apiClient.post<{ message: string }>('/admin/users/update-status', { 
      user_id: userId, 
      is_active: isActive 
    }),
  
  updateUser: (userId: number, email: string, firstName: string, lastName: string) =>
    apiClient.post<{ message: string }>('/admin/users/update', {
      user_id: userId,
      email,
      first_name: firstName,
      last_name: lastName
    }),
  
  setUserPassword: (userId: number, password: string) =>
    apiClient.post<{ message: string }>('/admin/users/set-password', {
      user_id: userId,
      password
    }),
  
  deleteUser: (userId: number) =>
    apiClient.post<{ message: string }>('/admin/users/delete', {
      user_id: userId
    }),
  
  // Role management
  listRoles: () =>
    apiClient.get<Role[]>('/admin/roles/list'),
  
  // Audit logs
  listAuditLogs: (params: AuditLogListParams = {}) => {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.user_id) queryParams.append('user_id', params.user_id.toString());
    if (params.action) queryParams.append('action', params.action);
    if (params.resource) queryParams.append('resource', params.resource);
    if (params.sort_by) queryParams.append('sort_by', params.sort_by);
    if (params.sort_order) queryParams.append('sort_order', params.sort_order);
    
    return apiClient.get<PaginatedResponse<AuditLog>>(`/admin/audit-logs/list?${queryParams.toString()}`);
  },
  
  // Session management
  getAllSessions: () =>
    apiClient.get<Session[]>('/admin/sessions'),
  
  deleteUserSession: (sessionId: string) =>
    apiClient.delete<{ message: string }>(`/admin/sessions/delete?session_id=${sessionId}`),
  
  deleteAllUserSessions: (userId: number) =>
    apiClient.delete<{ message: string }>(`/admin/sessions/delete-all?user_id=${userId}`),
};

export default adminApi;
