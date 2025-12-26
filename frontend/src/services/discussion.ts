import api from './api';

export interface DiscussionCategoryResult {
  id: number;
  discussion_result_id: number;
  category_id: number;
  category_name: string;
  user_level_id?: number;
  user_level_name?: string;
  reviewer_level_id: number;
  reviewer_level_name: string;
  reviewer_level_number: number;
  justification?: string;
  is_override: boolean;
}

export interface DiscussionReviewer {
  id: number;
  discussion_result_id: number;
  reviewer_user_id: number;
  reviewer_name: string;
}

export interface DiscussionConfirmation {
  id: number;
  assessment_id: number;
  user_id: number;
  user_name?: string;
  user_type: 'reviewer' | 'owner';
  confirmed_at: string;
  created_at: string;
  updated_at: string;
}

export interface DiscussionResult {
  id: number;
  assessment_id: number;
  weighted_overall_level_number: number;
  weighted_overall_level_id: number;
  weighted_overall_level_name: string;
  final_comment: string;
  discussion_note?: string;
  user_approved_at?: string;
  created_at: string;
  updated_at: string;
  category_results: DiscussionCategoryResult[];
  reviewers: DiscussionReviewer[];
  confirmations: DiscussionConfirmation[];
}

class DiscussionService {
  async getDiscussionResult(assessmentId: number): Promise<DiscussionResult> {
    const response = await api.get<DiscussionResult>(`/discussion/${assessmentId}`);
    return response;
  }

  async updateDiscussionNote(assessmentId: number, note: string): Promise<void> {
    await api.put(`/discussion/${assessmentId}/note`, { note });
  }

  async createConfirmation(assessmentId: number, userType: 'reviewer' | 'owner'): Promise<void> {
    await api.post(`/discussion/${assessmentId}/confirm`, { user_type: userType });
  }

  async archiveAssessment(assessmentId: number): Promise<void> {
    await api.post(`/assessments/${assessmentId}/archive`);
  }
}

export default new DiscussionService();
