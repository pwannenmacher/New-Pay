import api from './api';
import type { SelfAssessment, CriteriaCatalog } from '../types';

export const selfAssessmentService = {
  // Get catalogs available for creating self-assessments
  getActiveCatalogs: async (): Promise<CriteriaCatalog[]> => {
    try {
      return await api.get<CriteriaCatalog[]>('/self-assessments/active-catalogs');
    } catch (error) {
      console.error('Error fetching active catalogs:', error);
      throw error;
    }
  },

  // Create a new self-assessment
  createSelfAssessment: async (catalogId: number): Promise<SelfAssessment> => {
    try {
      return await api.post<SelfAssessment>(`/self-assessments/catalog/${catalogId}`);
    } catch (error) {
      console.error('Error creating self-assessment:', error);
      throw error;
    }
  },

  // Get current user's self-assessments
  getMySelfAssessments: async (): Promise<SelfAssessment[]> => {
    try {
      return await api.get<SelfAssessment[]>('/self-assessments/my');
    } catch (error) {
      console.error('Error fetching my self-assessments:', error);
      throw error;
    }
  },

  // Get visible self-assessments based on role
  getVisibleSelfAssessments: async (): Promise<SelfAssessment[]> => {
    try {
      return await api.get<SelfAssessment[]>('/self-assessments');
    } catch (error) {
      console.error('Error fetching visible self-assessments:', error);
      throw error;
    }
  },

  // Get a specific self-assessment
  getSelfAssessment: async (id: number): Promise<SelfAssessment> => {
    try {
      return await api.get<SelfAssessment>(`/self-assessments/${id}`);
    } catch (error) {
      console.error('Error fetching self-assessment:', error);
      throw error;
    }
  },

  // Update self-assessment status
  updateStatus: async (id: number, status: string): Promise<void> => {
    try {
      await api.put<void>(`/self-assessments/${id}/status`, { status });
    } catch (error) {
      console.error('Error updating self-assessment status:', error);
      throw error;
    }
  },
};
