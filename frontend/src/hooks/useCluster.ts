import { useEffect } from 'react';
import { useDashboardStore } from '../lib/store';
import { clusterApi } from '../lib/api';

export function useCluster() {
  const { clusterStatus, setClusterStatus } = useDashboardStore();

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const status = await clusterApi.getStatus();
        setClusterStatus(status);
      } catch (err) {
        setClusterStatus({
          connected: false,
          arm_nodes: 0,
        });
        console.error('Failed to fetch cluster status:', err);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 30000);
    return () => clearInterval(interval);
  }, [setClusterStatus]);

  return clusterStatus;
}
