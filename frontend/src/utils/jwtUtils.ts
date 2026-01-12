/**
 * Decodes a JWT token without verifying the signature
 * This is safe for reading expiration times on the client side
 */
export interface JWTPayload {
  exp?: number; // Expiration time (Unix timestamp)
  iat?: number; // Issued at (Unix timestamp)
  sub?: string; // Subject (usually user ID)
  email?: string;
  [key: string]: unknown;
}

export function decodeJWT(token: string): JWTPayload | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }

    const payload = parts[1];
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded);
  } catch (error) {
    console.error('Failed to decode JWT:', error);
    return null;
  }
}

/**
 * Checks if a JWT token is expired
 */
export function isTokenExpired(token: string): boolean {
  const payload = decodeJWT(token);
  if (!payload || !payload.exp) {
    return true;
  }

  const now = Math.floor(Date.now() / 1000);
  return payload.exp < now;
}

/**
 * Gets the expiration time of a JWT token in milliseconds
 */
export function getTokenExpirationTime(token: string): number | null {
  const payload = decodeJWT(token);
  if (!payload || !payload.exp) {
    return null;
  }

  return payload.exp * 1000; // Convert to milliseconds
}

/**
 * Calculates how many milliseconds until the token should be refreshed
 * Returns the time until 1 minute before expiration
 */
export function getTimeUntilRefresh(token: string): number | null {
  const expirationTime = getTokenExpirationTime(token);
  if (!expirationTime) {
    return null;
  }

  const now = Date.now();
  const refreshTime = expirationTime - 60 * 1000; // 1 minute before expiration
  const timeUntilRefresh = refreshTime - now;

  // If already past refresh time, refresh immediately
  return Math.max(0, timeUntilRefresh);
}
