import apiClient from './api';
import type { UserWithRoles, Role, AuditLog, AssignRoleRequest, RemoveRoleRequest } from '../types';

export const adminApi = {
  // User management
  getUser: (userId: number) =>
    apiClient.get<UserWithRoles>(`/admin/users/get?id=${userId}`),
  
  listUsers: (page = 1, limit = 20) =>
    apiClient.get<UserWithRoles[]>(`/admin/users/list?page=${page}&limit=${limit}`),
  
  assignRole: (data: AssignRoleRequest) =>
    apiClient.post<{ message: string }>('/admin/users/assign-role', data),
  
  removeRole: (data: RemoveRoleRequest) =>
    apiClient.post<{ message: string }>('/admin/users/remove-role', data),
  
  // Role management
  listRoles: () =>
    apiClient.get<Role[]>('/admin/roles/list'),
  
  // Audit logs
  listAuditLogs: (page = 1, limit = 50) =>
    apiClient.get<AuditLog[]>(`/admin/audit-logs/list?page=${page}&limit=${limit}`),
};

export default adminApi;
