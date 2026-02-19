import { useEffect, useState } from 'react';
import { Settings, CheckCircle, XCircle, AlertTriangle, Trash2 } from 'lucide-react';
import OperatorInstallWizard from '../components/OperatorInstallWizard';
import { useDashboardStore } from '../lib/store';
import { operatorApi } from '../lib/api';
import type { OperatorStatus } from '../types/api';

export default function SettingsPage() {
  const { clusterStatus, updateCounter, uninstallProgress, clearUninstallProgress } = useDashboardStore();
  const [operatorStatus, setOperatorStatus] = useState<OperatorStatus | null>(null);
  const [confirmUninstall, setConfirmUninstall] = useState(false);
  const [uninstalling, setUninstalling] = useState(false);

  useEffect(() => {
    if (clusterStatus?.connected) {
      operatorApi.getStatus().then(setOperatorStatus).catch(console.error);
    }
  }, [clusterStatus?.connected, updateCounter]);

  // When uninstall completes, update local state
  useEffect(() => {
    const last = uninstallProgress[uninstallProgress.length - 1];
    if (last?.done) {
      setUninstalling(false);
      if (!last.error) {
        setOperatorStatus({ installed: false });
      }
    }
  }, [uninstallProgress]);

  const handleUninstall = async () => {
    setUninstalling(true);
    setConfirmUninstall(false);
    clearUninstallProgress();
    try {
      await operatorApi.uninstall();
    } catch (err) {
      setUninstalling(false);
      console.error('Uninstall request failed:', err);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
        <p className="text-sm text-gray-500 mt-1">Configure the dashboard and manage the operator</p>
      </div>

      {/* Cluster Info */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="font-semibold text-gray-900 flex items-center gap-2">
            <Settings className="h-4 w-4" />
            Cluster Configuration
          </h3>
        </div>
        <div className="p-6 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wider">Server</label>
              <p className="text-sm text-gray-900 font-mono mt-1">
                {clusterStatus?.server_url || 'Not connected'}
              </p>
            </div>
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wider">Version</label>
              <p className="text-sm text-gray-900 mt-1">
                {clusterStatus?.server_version || 'Unknown'}
              </p>
            </div>
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wider">Platform</label>
              <p className="text-sm text-gray-900 mt-1">
                {clusterStatus?.platform || 'Unknown'}
              </p>
            </div>
            <div>
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wider">Architecture</label>
              <p className="text-sm text-gray-900 mt-1">
                {clusterStatus?.architecture || 'Unknown'}
                {clusterStatus?.arm_nodes ? ` (${clusterStatus.arm_nodes} ARM nodes)` : ''}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Operator Status */}
      <div className="card">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="font-semibold text-gray-900">Operator Status</h3>
        </div>
        <div className="p-6">
          {operatorStatus ? (
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                {operatorStatus.installed ? (
                  <CheckCircle className="h-5 w-5 text-emerald-500" />
                ) : (
                  <XCircle className="h-5 w-5 text-gray-400" />
                )}
                <div>
                  <p className="font-medium text-sm text-gray-900">
                    {operatorStatus.installed ? 'Installed' : 'Not Installed'}
                  </p>
                  {operatorStatus.version && (
                    <p className="text-xs text-gray-500">{operatorStatus.version}</p>
                  )}
                </div>
              </div>

              {operatorStatus.pods && operatorStatus.pods.length > 0 && (
                <div>
                  <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Pods</h4>
                  <div className="space-y-1">
                    {operatorStatus.pods.map(pod => (
                      <div key={pod.name} className="flex items-center gap-2 text-sm">
                        <span className={`inline-block h-2 w-2 rounded-full ${
                          pod.ready ? 'bg-emerald-500' : 'bg-amber-500'
                        }`} />
                        <span className="font-mono text-xs text-gray-700">{pod.name}</span>
                        <span className="text-xs text-gray-500">{pod.phase}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {operatorStatus.profile_bundles && operatorStatus.profile_bundles.length > 0 && (
                <div>
                  <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">ProfileBundles</h4>
                  <div className="space-y-1">
                    {operatorStatus.profile_bundles.map(pb => (
                      <div key={pb.name} className="flex items-center gap-2 text-sm">
                        <span className={`badge ${
                          pb.data_stream_status === 'VALID' ? 'badge-pass' : 'badge-manual'
                        }`}>
                          {pb.data_stream_status}
                        </span>
                        <span className="text-gray-700">{pb.name}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-gray-500">Loading operator status...</p>
          )}
        </div>
      </div>

      {/* Uninstall section (show when installed) */}
      {operatorStatus?.installed && clusterStatus?.connected && (
        <div className="card border-red-200">
          <div className="px-6 py-4 border-b border-red-200 bg-red-50">
            <h3 className="font-semibold text-red-900 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Danger Zone
            </h3>
          </div>
          <div className="p-6">
            {!uninstalling && uninstallProgress.length === 0 && (
              <>
                <p className="text-sm text-gray-600 mb-4">
                  Uninstalling the Compliance Operator will remove all compliance resources,
                  scan results, and the operator itself from the cluster.
                </p>
                {!confirmUninstall ? (
                  <button
                    className="btn px-4 py-2 bg-red-600 text-white hover:bg-red-700"
                    onClick={() => setConfirmUninstall(true)}
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Uninstall Operator
                  </button>
                ) : (
                  <div className="bg-red-50 border border-red-300 rounded-lg p-4">
                    <p className="text-sm text-red-800 font-medium mb-3">
                      This action cannot be undone. All compliance data will be permanently deleted.
                    </p>
                    <div className="flex gap-2">
                      <button
                        className="btn px-4 py-2 bg-red-600 text-white hover:bg-red-700"
                        onClick={handleUninstall}
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        Confirm Uninstall
                      </button>
                      <button
                        className="btn btn-secondary px-4 py-2"
                        onClick={() => setConfirmUninstall(false)}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                )}
              </>
            )}

            {/* Uninstall progress */}
            {(uninstalling || uninstallProgress.length > 0) && (
              <div className="space-y-2">
                {uninstallProgress.map((p, i) => (
                  <div key={i} className="flex items-center gap-2 text-sm">
                    {p.error ? (
                      <XCircle className="h-4 w-4 text-red-500 shrink-0" />
                    ) : p.done ? (
                      <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" />
                    ) : (
                      <div className="h-4 w-4 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 shrink-0" />
                    )}
                    <span className={p.error ? 'text-red-700' : 'text-gray-700'}>{p.message}</span>
                  </div>
                ))}
                {uninstalling && (
                  <div className="flex items-center gap-2 text-sm text-gray-500 mt-2">
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600" />
                    Uninstalling...
                  </div>
                )}
                {!uninstalling && uninstallProgress.length > 0 && (
                  <button
                    className="btn btn-secondary text-xs mt-3"
                    onClick={clearUninstallProgress}
                  >
                    Clear
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Install Wizard (only show if not installed) */}
      {operatorStatus && !operatorStatus.installed && clusterStatus?.connected && (
        <OperatorInstallWizard />
      )}
    </div>
  );
}
