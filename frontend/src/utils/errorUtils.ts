import type { ApiError } from '../types';

/**
 * Extract an error message from an unknown caught error.
 * Handles ApiError objects thrown by the API client, standard Error instances,
 * and arbitrary values.
 */
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error !== null && typeof error === 'object' && 'error' in error) {
    return (error as ApiError).error || fallback;
  }
  if (error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
}
