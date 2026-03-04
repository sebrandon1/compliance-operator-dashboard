import type { Severity, CheckStatus } from '../types/api';

export const severityOrder: Record<string, number> = { high: 0, medium: 1, low: 2 };

export function severityBadgeClass(severity: Severity): string {
  switch (severity) {
    case 'high': return 'badge-high';
    case 'medium': return 'badge-medium';
    case 'low': return 'badge-low';
    default: return 'badge-skip';
  }
}

export function statusBadgeClass(status: CheckStatus): string {
  switch (status) {
    case 'PASS': return 'badge-pass';
    case 'FAIL': return 'badge-fail';
    case 'MANUAL': return 'badge-manual';
    default: return 'badge-skip';
  }
}
