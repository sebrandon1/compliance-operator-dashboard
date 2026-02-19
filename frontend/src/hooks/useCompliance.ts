import { useEffect, useCallback } from 'react';
import { useDashboardStore } from '../lib/store';
import { resultsApi } from '../lib/api';

export function useCompliance() {
  const { complianceData, setComplianceData, updateCounter } = useDashboardStore();

  const refresh = useCallback(async () => {
    try {
      const data = await resultsApi.getAll();
      setComplianceData(data);
    } catch (err) {
      console.error('Failed to fetch compliance data:', err);
    }
  }, [setComplianceData]);

  useEffect(() => {
    // Debounce refetches from rapid WebSocket events
    const timer = setTimeout(() => {
      refresh();
    }, updateCounter === 0 ? 0 : 2000);
    return () => clearTimeout(timer);
  }, [refresh, updateCounter]);

  return { complianceData, refresh };
}
