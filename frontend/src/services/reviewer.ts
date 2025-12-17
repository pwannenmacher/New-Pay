import apiClient from './api';

export interface ReviewerResponse {
  id?: number;
  assessment_id: number;
  category_id: number;
  reviewer_user_id: number;
  path_id: number;
  level_id: number;
  justification?: string;
  created_at?: string;
  updated_at?: string;
}

export interface ReviewerResponseInput {
  category_id: number;
  path_id: number;
  level_id: number;
  justification?: string;
}

export interface ReviewCompletionStatus {
  total_reviewers: number;
  complete_reviews: number;
  can_consolidate: boolean;
  reviewers_with_complete_reviews: Array<{
    reviewer_id: number;
    reviewer_name: string;
    completed_at: string;
  }>;
}

/**
 * Reviewer service for managing individual reviewer assessments
 */
const reviewerService = {
  /**
   * Get reviewer's own responses for an assessment
   */
  getResponses: (assessmentId: number) =>
    apiClient.get<ReviewerResponse[]>(`/review/assessment/${assessmentId}/responses`),

  /**
   * Create or update a reviewer response for a category
   */
  saveResponse: (assessmentId: number, data: ReviewerResponseInput) =>
    apiClient.post<ReviewerResponse>(`/review/assessment/${assessmentId}/responses`, data),

  /**
   * Delete a reviewer response for a category
   */
  deleteResponse: (assessmentId: number, categoryId: number) =>
    apiClient.delete<{ message: string }>(`/review/assessment/${assessmentId}/responses/${categoryId}`),

  /**
   * Complete the review (mark all categories as reviewed)
   */
  completeReview: (assessmentId: number, newStatus: string) =>
    apiClient.post<{ message: string; assessment: any }>(`/review/assessment/${assessmentId}/complete`, {
      new_status: newStatus,
    }),

  /**
   * Get completion status (how many reviewers have completed their review)
   */
  getCompletionStatus: (assessmentId: number) =>
    apiClient.get<ReviewCompletionStatus>(`/review/assessment/${assessmentId}/completion-status`),
};

export default reviewerService;
