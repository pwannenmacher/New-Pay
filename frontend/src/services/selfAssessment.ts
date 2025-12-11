import api from './api';
import type { 
  SelfAssessment, 
  CriteriaCatalog, 
  AssessmentResponse, 
  AssessmentResponseWithDetails, 
  AssessmentCompleteness 
} from '../types';

export const selfAssessmentService = {
  // Get catalogs available for creating self-assessments
  getActiveCatalogs: async (): Promise<CriteriaCatalog[]> => {
    try {
      return await api.get<CriteriaCatalog[]>('/self-assessments/active-catalogs');
    } catch (error) {
      console.error('Error fetching active catalogs:', error);
      return [];
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
      return [];
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

  // Get all self-assessments with filters (admin only)
  getAllSelfAssessmentsAdmin: async (filters?: {
    status?: string;
    username?: string;
    from_date?: string;
    to_date?: string;
  }): Promise<SelfAssessment[]> => {
    try {
      const params = new URLSearchParams();
      if (filters?.status) params.append('status', filters.status);
      if (filters?.username) params.append('username', filters.username);
      if (filters?.from_date) params.append('from_date', filters.from_date);
      if (filters?.to_date) params.append('to_date', filters.to_date);
      
      const queryString = params.toString();
      const url = `/admin/self-assessments${queryString ? `?${queryString}` : ''}`;
      return await api.get<SelfAssessment[]>(url);
    } catch (error) {
      console.error('Error fetching all self-assessments:', error);
      throw error;
    }
  },

  // Delete self-assessment (admin only)
  deleteSelfAssessment: async (id: number): Promise<void> => {
    try {
      await api.delete<void>(`/admin/self-assessments/${id}`);
    } catch (error) {
      console.error('Error deleting self-assessment:', error);
      throw error;
    }
  },

  // Get responses for an assessment
  getResponses: async (assessmentId: number): Promise<AssessmentResponseWithDetails[]> => {
    try {
      return await api.get<AssessmentResponseWithDetails[]>(`/self-assessments/${assessmentId}/responses`);
    } catch (error) {
      console.error('Error fetching responses:', error);
      throw error;
    }
  },

  // Save or update a response
  saveResponse: async (assessmentId: number, response: Omit<AssessmentResponse, 'id' | 'assessment_id' | 'created_at' | 'updated_at'>): Promise<AssessmentResponse> => {
    try {
      return await api.post<AssessmentResponse>(`/self-assessments/${assessmentId}/responses`, response);
    } catch (error) {
      console.error('Error saving response:', error);
      throw error;
    }
  },

  // Delete a response
  deleteResponse: async (assessmentId: number, categoryId: number): Promise<void> => {
    try {
      await api.delete<void>(`/self-assessments/${assessmentId}/responses/${categoryId}`);
    } catch (error) {
      console.error('Error deleting response:', error);
      throw error;
    }
  },

  // Get completeness status
  getCompleteness: async (assessmentId: number): Promise<AssessmentCompleteness> => {
    try {
      return await api.get<AssessmentCompleteness>(`/self-assessments/${assessmentId}/completeness`);
    } catch (error) {
      console.error('Error fetching completeness:', error);
      throw error;
    }
  },

  // Submit assessment for review
  submitAssessment: async (assessmentId: number): Promise<void> => {
    try {
      await api.put<void>(`/self-assessments/${assessmentId}/submit`, {});
    } catch (error) {
      console.error('Error submitting assessment:', error);
      throw error;
    }
  },
};
