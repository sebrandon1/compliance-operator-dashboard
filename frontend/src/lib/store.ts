import { create } from 'zustand';
import type {
  ClusterStatus,
  OperatorStatus,
  ComplianceData,
  InstallProgress,
  WSMessage,
} from '../types/api';

interface DashboardState {
  // Connection state
  wsConnected: boolean;
  setWSConnected: (connected: boolean) => void;

  // Cluster state
  clusterStatus: ClusterStatus | null;
  setClusterStatus: (status: ClusterStatus) => void;

  // Operator state
  operatorStatus: OperatorStatus | null;
  setOperatorStatus: (status: OperatorStatus) => void;

  // Compliance data
  complianceData: ComplianceData | null;
  setComplianceData: (data: ComplianceData) => void;

  // Install progress
  installProgress: InstallProgress[];
  addInstallProgress: (progress: InstallProgress) => void;
  clearInstallProgress: () => void;

  // Uninstall progress
  uninstallProgress: InstallProgress[];
  addUninstallProgress: (progress: InstallProgress) => void;
  clearUninstallProgress: () => void;

  // Live updates counter (triggers refetches)
  updateCounter: number;
  incrementUpdateCounter: () => void;

  // Handle incoming WebSocket message
  handleWSMessage: (msg: WSMessage) => void;
}

export const useDashboardStore = create<DashboardState>((set) => ({
  wsConnected: false,
  setWSConnected: (connected) => set({ wsConnected: connected }),

  clusterStatus: null,
  setClusterStatus: (status) => set({ clusterStatus: status }),

  operatorStatus: null,
  setOperatorStatus: (status) => set({ operatorStatus: status }),

  complianceData: null,
  setComplianceData: (data) => set({ complianceData: data }),

  installProgress: [],
  addInstallProgress: (progress) =>
    set((state) => ({
      installProgress: [...state.installProgress, progress],
    })),
  clearInstallProgress: () => set({ installProgress: [] }),

  uninstallProgress: [],
  addUninstallProgress: (progress) =>
    set((state) => ({
      uninstallProgress: [...state.uninstallProgress, progress],
    })),
  clearUninstallProgress: () => set({ uninstallProgress: [] }),

  updateCounter: 0,
  incrementUpdateCounter: () =>
    set((state) => ({ updateCounter: state.updateCounter + 1 })),

  handleWSMessage: (msg: WSMessage) => {
    switch (msg.type) {
      case 'cluster_status':
        set({ clusterStatus: msg.payload as ClusterStatus });
        break;

      case 'operator_status':
        set({ operatorStatus: msg.payload as OperatorStatus });
        break;

      case 'install_progress': {
        const progress = msg.payload as InstallProgress;
        set((state) => ({
          installProgress: [...state.installProgress, progress],
          // When install completes (or fails), trigger refetches across the app
          updateCounter: progress.done ? state.updateCounter + 1 : state.updateCounter,
        }));
        break;
      }

      case 'uninstall_progress': {
        const progress = msg.payload as InstallProgress;
        set((state) => ({
          uninstallProgress: [...state.uninstallProgress, progress],
          updateCounter: progress.done ? state.updateCounter + 1 : state.updateCounter,
          // Clear operator status when uninstall completes successfully
          ...(progress.done && !progress.error ? { operatorStatus: { installed: false } as OperatorStatus } : {}),
        }));
        break;
      }

      case 'check_result':
      case 'remediation':
      case 'scan_status':
      case 'remediation_result':
        // Trigger a refetch of compliance data on any watch event
        set((state) => ({ updateCounter: state.updateCounter + 1 }));
        break;

      case 'error':
        console.error('WebSocket error:', msg.payload);
        break;
    }
  },
}));
