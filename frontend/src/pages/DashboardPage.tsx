import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Play, FileSearch, CheckCircle, XCircle, AlertTriangle, HelpCircle } from 'lucide-react';
import ConnectionBanner from '../components/ConnectionBanner';
import SeverityCard from '../components/SeverityCard';
import { useCompliance } from '../hooks/useCompliance';
import { useDashboardStore } from '../lib/store';
import { operatorApi, scanApi } from '../lib/api';
import type { OperatorStatus } from '../types/api';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { complianceData } = useCompliance();
  const { clusterStatus } = useDashboardStore();
  const [operatorStatus, setOperatorStatus] = useState<OperatorStatus | null>(null);
  const [creatingaScan, setCreatingScan] = useState(false);

  useEffect(() => {
    if (!clusterStatus?.connected) return;
    operatorApi.getStatus().then(setOperatorStatus).catch(console.error);
  }, [clusterStatus?.connected]);

  const handleRunScan = async () => {
    setCreatingScan(true);
    try {
      await scanApi.create({ name: 'cis-scan', profile: 'ocp4-cis' });
    } catch (err) {
      console.error('Failed to create scan:', err);
    } finally {
      setCreatingScan(false);
    }
  };

  const summary = complianceData?.summary;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-sm text-gray-500 mt-1">Compliance Operator overview and quick actions</p>
      </div>

      {/* Connection Banner */}
      <ConnectionBanner />

      {/* Operator Status */}
      {operatorStatus && (
        <div className="card p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className={`h-8 w-8 rounded-lg flex items-center justify-center ${
                operatorStatus.installed ? 'bg-emerald-100' : 'bg-gray-100'
              }`}>
                {operatorStatus.installed ? (
                  <CheckCircle className="h-5 w-5 text-emerald-600" />
                ) : (
                  <XCircle className="h-5 w-5 text-gray-400" />
                )}
              </div>
              <div>
                <p className="font-medium text-sm text-gray-900">
                  Compliance Operator {operatorStatus.installed ? 'Installed' : 'Not Installed'}
                </p>
                {operatorStatus.version && (
                  <p className="text-xs text-gray-500">{operatorStatus.version}</p>
                )}
              </div>
            </div>
            {operatorStatus.profile_bundles && operatorStatus.profile_bundles.length > 0 && (
              <div className="flex gap-2">
                {operatorStatus.profile_bundles.map(pb => (
                  <span
                    key={pb.name}
                    className={`badge ${pb.data_stream_status === 'VALID' ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}
                  >
                    {pb.name}: {pb.data_stream_status}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Severity Cards */}
      {complianceData && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <SeverityCard
            severity="high"
            failCount={complianceData.remediations.high?.length ?? 0}
            passCount={complianceData.passing_checks.high?.length ?? 0}
            total={(complianceData.remediations.high?.length ?? 0) + (complianceData.passing_checks.high?.length ?? 0)}
          />
          <SeverityCard
            severity="medium"
            failCount={complianceData.remediations.medium?.length ?? 0}
            passCount={complianceData.passing_checks.medium?.length ?? 0}
            total={(complianceData.remediations.medium?.length ?? 0) + (complianceData.passing_checks.medium?.length ?? 0)}
          />
          <SeverityCard
            severity="low"
            failCount={complianceData.remediations.low?.length ?? 0}
            passCount={complianceData.passing_checks.low?.length ?? 0}
            total={(complianceData.remediations.low?.length ?? 0) + (complianceData.passing_checks.low?.length ?? 0)}
          />
        </div>
      )}

      {/* Summary Bar */}
      {summary && summary.total_checks > 0 && (
        <div className="card p-4">
          <h3 className="font-medium text-sm text-gray-900 mb-3">Check Summary</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="flex items-center gap-2">
              <CheckCircle className="h-4 w-4 text-emerald-500" />
              <span className="text-sm text-gray-600">Pass</span>
              <span className="font-bold text-gray-900">{summary.passing}</span>
            </div>
            <div className="flex items-center gap-2">
              <XCircle className="h-4 w-4 text-red-500" />
              <span className="text-sm text-gray-600">Fail</span>
              <span className="font-bold text-gray-900">{summary.failing}</span>
            </div>
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-amber-500" />
              <span className="text-sm text-gray-600">Manual</span>
              <span className="font-bold text-gray-900">{summary.manual}</span>
            </div>
            <div className="flex items-center gap-2">
              <HelpCircle className="h-4 w-4 text-gray-400" />
              <span className="text-sm text-gray-600">Skipped</span>
              <span className="font-bold text-gray-900">{summary.skipped}</span>
            </div>
          </div>

          {/* Overall progress bar */}
          <div className="mt-4 h-3 rounded-full bg-gray-200 overflow-hidden flex">
            <div
              className="bg-emerald-500 transition-all"
              style={{ width: `${(summary.passing / summary.total_checks) * 100}%` }}
            />
            <div
              className="bg-red-500 transition-all"
              style={{ width: `${(summary.failing / summary.total_checks) * 100}%` }}
            />
            <div
              className="bg-amber-400 transition-all"
              style={{ width: `${(summary.manual / summary.total_checks) * 100}%` }}
            />
          </div>
          <p className="text-xs text-gray-500 mt-1.5">
            {Math.round((summary.passing / summary.total_checks) * 100)}% compliance rate across {summary.total_checks} checks
          </p>
        </div>
      )}

      {/* Quick Actions */}
      <div className="flex gap-3">
        <button
          className="btn btn-primary"
          onClick={handleRunScan}
          disabled={creatingaScan || !clusterStatus?.connected}
        >
          {creatingaScan ? (
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent mr-2" />
          ) : (
            <Play className="h-4 w-4 mr-2" />
          )}
          Run Scan
        </button>
        <button
          className="btn btn-secondary"
          onClick={() => navigate('/results')}
        >
          <FileSearch className="h-4 w-4 mr-2" />
          View Results
        </button>
      </div>

      {/* No data state */}
      {!complianceData && clusterStatus?.connected && (
        <div className="card p-8 text-center">
          <FileSearch className="h-12 w-12 text-gray-300 mx-auto mb-3" />
          <p className="text-gray-500">No compliance data available yet.</p>
          <p className="text-sm text-gray-400 mt-1">Run a scan to see results.</p>
        </div>
      )}
    </div>
  );
}
