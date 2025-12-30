import api from './api';

export interface ConsolidationAveragedApproval {
  id: number;
  assessment_id: number;
  category_id: number;
  approved_by_user_id: number;
  approved_by_name: string;
  approved_at: string;
}

export interface AveragedReviewerResponse {
  category_id: number;
  category_name: string;
  category_sort_order: number;
  average_level_number: number;
  average_level_name: string;
  reviewer_count: number;
  reviewer_justifications?: string[];
  approvals?: ConsolidationAveragedApproval[];
  approval_count?: number;
  is_approved?: boolean;
}

export interface ConsolidationOverrideApproval {
  id: number;
  override_id: number;
  approved_by_user_id: number;
  approved_by_name: string;
  approved_at: string;
}

export interface ConsolidationOverride {
  id?: number;
  assessment_id: number;
  category_id: number;
  path_id: number;
  level_id: number;
  justification: string;
  created_by_user_id?: number;
  created_at?: string;
  updated_at?: string;
  approvals?: ConsolidationOverrideApproval[];
  approval_count?: number;
  is_approved?: boolean;
}

export interface CategoryDiscussionComment {
  id: number;
  assessment_id: number;
  category_id: number;
  comment: string;
  encrypted_comment_id?: number;
  created_by_user_id: number;
  created_at: string;
  updated_at: string;
}

export interface ConsolidationData {
  assessment: any; // SelfAssessment type
  user_responses: any[]; // AssessmentResponseWithDetails[]
  averaged_responses: AveragedReviewerResponse[];
  overrides: ConsolidationOverride[];
  catalog: any; // CatalogWithDetails type
  current_user_responses?: any[]; // Current user's own reviewer responses
  final_consolidation?: FinalConsolidation;
  all_categories_approved?: boolean;
  category_discussion_comments?: CategoryDiscussionComment[];
}

export interface FinalConsolidationApproval {
  id: number;
  assessment_id: number;
  approved_by_user_id: number;
  approved_by_name: string;
  approved_at: string;
}

export interface FinalConsolidation {
  id: number;
  assessment_id: number;
  comment: string;
  encrypted_comment_id?: number;
  created_by_user_id: number;
  created_at: string;
  updated_at: string;
  approvals?: FinalConsolidationApproval[];
  approval_count?: number;
  required_approvals?: number;
  is_fully_approved?: boolean;
}

class ConsolidationService {
  async getConsolidationData(assessmentId: number): Promise<ConsolidationData> {
    const response = await api.get(`/review/consolidation/${assessmentId}`);
    return response as ConsolidationData;
  }

  async createOrUpdateOverride(
    assessmentId: number,
    override: ConsolidationOverride
  ): Promise<ConsolidationOverride> {
    const response = await api.post(`/review/consolidation/${assessmentId}/override`, override);
    return response as ConsolidationOverride;
  }

  async approveOverride(assessmentId: number, categoryId: number): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/override/${categoryId}/approve`, {});
  }

  async revokeOverrideApproval(assessmentId: number, categoryId: number): Promise<void> {
    await api.delete(`/review/consolidation/${assessmentId}/override/${categoryId}/approve`);
  }

  async deleteOverride(assessmentId: number, categoryId: number): Promise<void> {
    await api.delete(`/review/consolidation/${assessmentId}/override/${categoryId}`);
  }

  async approveAveragedResponse(assessmentId: number, categoryId: number): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/averaged/${categoryId}/approve`, {});
  }

  async revokeAveragedApproval(assessmentId: number, categoryId: number): Promise<void> {
    await api.delete(`/review/consolidation/${assessmentId}/averaged/${categoryId}/approve`);
  }

  async saveFinalConsolidation(assessmentId: number, comment: string): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/final`, { comment });
  }

  async approveFinalConsolidation(assessmentId: number): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/final/approve`, {});
  }

  async revokeFinalApproval(assessmentId: number): Promise<void> {
    await api.delete(`/review/consolidation/${assessmentId}/final/approve`);
  }

  async saveCategoryDiscussionComment(
    assessmentId: number,
    categoryId: number,
    comment: string
  ): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/category/${categoryId}/comment`, {
      comment,
    });
  }

  async regenerateProposals(assessmentId: number): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/regenerate-proposals`, {});
  }

  async generateFinalProposal(assessmentId: number): Promise<void> {
    await api.post(`/review/consolidation/${assessmentId}/generate-final-proposal`, {});
  }
}

export default new ConsolidationService();
