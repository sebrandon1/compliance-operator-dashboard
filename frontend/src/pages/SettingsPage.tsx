import { useEffect, useState } from 'react';
import { Settings, CheckCircle, XCircle } from 'lucide-react';
import OperatorInstallWizard from '../components/OperatorInstallWizard';
import { useDashboardStore } from '../lib/store';
import { operatorApi } from '../lib/api';
import type { OperatorStatus } from '../types/api';

export default function SettingsPage() {
  const { clusterStatus, updateCounter } = useDashboardStore();
  const [operatorStatus, setOperatorStatus] = useState<OperatorStatus | null>(null);

  useEffect(() => {
    if (clusterStatus?.connected) {
      operatorApi.getStatus().then(setOperatorStatus).catch(console.error);
    }
  }, [clusterStatus?.connected, updateCounter]);

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

      {/* Install Wizard (only show if not installed) */}
      {operatorStatus && !operatorStatus.installed && clusterStatus?.connected && (
        <OperatorInstallWizard />
      )}
    </div>
  );
}
