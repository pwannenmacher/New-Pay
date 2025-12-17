import apiClient from './api';
import type { CatalogWithDetails, CriteriaCatalog } from '../types';

/**
 * Public catalog service for accessing catalog information
 * Available to users and reviewers (read-only)
 */
const catalogService = {
  /**
   * Get all catalogs
   */
  listCatalogs: () => apiClient.get<CriteriaCatalog[]>('/catalogs'),

  /**
   * Get catalog with full details (categories, paths, levels, descriptions)
   */
  getCatalog: (catalogId: number) =>
    apiClient.get<CatalogWithDetails>(`/catalogs/${catalogId}`),
};

export default catalogService;
