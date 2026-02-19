import { Server, Wifi, WifiOff } from 'lucide-react';
import { useDashboardStore } from '../lib/store';

export default function ConnectionBanner() {
  const { clusterStatus, wsConnected } = useDashboardStore();

  if (!clusterStatus) {
    return (
      <div className="card p-4">
        <div className="flex items-center gap-3 text-gray-500">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600" />
          <span className="text-sm">Checking cluster connection...</span>
        </div>
      </div>
    );
  }

  return (
    <div className={`card p-4 ${clusterStatus.connected ? 'border-l-4 border-l-emerald-500' : 'border-l-4 border-l-red-500'}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Server className={`h-5 w-5 ${clusterStatus.connected ? 'text-emerald-600' : 'text-red-500'}`} />
          <div>
            <div className="flex items-center gap-2">
              <span className="font-medium text-sm">
                {clusterStatus.connected ? 'Connected' : 'Disconnected'}
              </span>
              {clusterStatus.server_version && (
                <span className="badge bg-gray-100 text-gray-700">
                  {clusterStatus.server_version}
                </span>
              )}
              {clusterStatus.platform && (
                <span className="badge bg-primary-100 text-primary-700">
                  {clusterStatus.platform}
                </span>
              )}
              {clusterStatus.architecture && (
                <span className="badge bg-gray-100 text-gray-600">
                  {clusterStatus.architecture}
                </span>
              )}
            </div>
            {clusterStatus.server_url && (
              <p className="text-xs text-gray-500 mt-0.5 font-mono">
                {clusterStatus.server_url}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2 text-xs text-gray-500">
          {wsConnected ? (
            <Wifi className="h-4 w-4 text-emerald-500" />
          ) : (
            <WifiOff className="h-4 w-4 text-red-400" />
          )}
          <span>{wsConnected ? 'Live' : 'Offline'}</span>
        </div>
      </div>
    </div>
  );
}
