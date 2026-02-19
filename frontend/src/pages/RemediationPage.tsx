import { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { Shield, Clock } from 'lucide-react';
import RemediationPanel from '../components/RemediationPanel';
import { remediationApi } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import type { RemediationInfo } from '../types/api';

export default function RemediationPage() {
  const [remediations, setRemediations] = useState<RemediationInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { updateCounter, clusterStatus } = useDashboardStore();

  const fetchRemediations = useCallback(async () => {
    try {
      const data = await remediationApi.list();
      setRemediations(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch remediations');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (clusterStatus?.connected) {
      // Debounce refetches from rapid WebSocket events
      const timer = setTimeout(() => {
        fetchRemediations();
      }, updateCounter === 0 ? 0 : 2000);
      return () => clearTimeout(timer);
    }
  }, [clusterStatus?.connected, updateCounter, fetchRemediations]);

  const notApplied = remediations.filter(r => !r.applied);
  const applied = remediations.filter(r => r.applied);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Remediation</h1>
          <p className="text-sm text-gray-500 mt-1">
            Apply compliance remediations to your cluster
          </p>
        </div>
        <div className="flex gap-2">
          <span className="badge bg-gray-100 text-gray-600">
            {notApplied.length} pending
          </span>
          <span className="badge bg-emerald-100 text-emerald-700">
            {applied.length} applied
          </span>
        </div>
      </div>

      {loading ? (
        <div className="card p-8 text-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 mx-auto mb-3" />
          <p className="text-sm text-gray-500">Loading remediations...</p>
        </div>
      ) : error ? (
        <div className="card border-red-200 bg-red-50 p-6 text-center">
          <p className="text-sm text-red-700">{error}</p>
          <button className="btn btn-secondary mt-3" onClick={fetchRemediations}>
            Retry
          </button>
        </div>
      ) : (
        <>
          {notApplied.length > 0 && (
            <div>
              <h2 className="font-semibold text-gray-900 mb-3">Pending Remediations</h2>
              <RemediationPanel remediations={notApplied} onApplied={fetchRemediations} />
            </div>
          )}

          {applied.length > 0 && (
            <div>
              <h2 className="font-semibold text-gray-900 mb-3">Applied Remediations</h2>
              <div className="card divide-y divide-gray-100">
                {applied.map(rem => {
                  const appliedAt = localStorage.getItem(`remediation-applied-${rem.name}`);
                  return (
                    <div key={rem.name} className="px-4 py-3 flex items-center gap-3">
                      <Shield className="h-4 w-4 text-emerald-500" />
                      <Link
                        to={`/remediation/${encodeURIComponent(rem.name)}`}
                        className="font-mono text-xs text-primary-600 hover:text-primary-800 hover:underline truncate"
                      >
                        {rem.name}
                      </Link>
                      <span className="badge bg-emerald-100 text-emerald-700">Applied</span>
                      {appliedAt && (
                        <span className="inline-flex items-center gap-1 text-xs text-gray-500">
                          <Clock className="h-3 w-3" />
                          {new Date(appliedAt).toLocaleString()}
                        </span>
                      )}
                      {rem.kind && <span className="badge bg-gray-100 text-gray-600">{rem.kind}</span>}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {remediations.length === 0 && (
            <div className="card p-12 text-center">
              <Shield className="h-12 w-12 text-gray-300 mx-auto mb-3" />
              <p className="text-gray-500">No remediations available.</p>
              <p className="text-sm text-gray-400 mt-1">Run a scan first to generate remediations.</p>
            </div>
          )}
        </>
      )}
    </div>
  );
}
