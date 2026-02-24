/**
 * Formats IP addresses with proxy information
 * Input: "192.168.1.1, 10.0.0.1, 172.16.0.1"
 * Output: "192.168.1.1 (proxied by 10.0.0.1, 172.16.0.1)"
 */
export function formatIPWithProxies(ipAddress: string | null | undefined): string {
  if (!ipAddress) return 'N/A';

  const ips = ipAddress.split(',').map((ip) => ip.trim());

  if (ips.length === 1) {
    return ips[0];
  }

  const clientIP = ips[0];
  const proxyIPs = ips.slice(1);

  return `${clientIP} (proxied by ${proxyIPs.join(', ')})`;
}

/**
 * Extracts only the client IP (first IP) from the address string
 * Input: "192.168.1.1, 10.0.0.1, 172.16.0.1"
 * Output: "192.168.1.1"
 */
export function getClientIP(ipAddress: string | null | undefined): string {
  if (!ipAddress) return 'N/A';

  const ips = ipAddress.split(',').map((ip) => ip.trim());
  return ips[0] || 'N/A';
}
