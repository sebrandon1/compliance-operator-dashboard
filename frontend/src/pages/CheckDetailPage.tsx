import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Shield, Wrench } from 'lucide-react';
import { resultsApi } from '../lib/api';
import type { CheckResultDetail, CheckStatus, Severity } from '../types/api';

function statusBadgeClass(status: CheckStatus): string {
  switch (status) {
    case 'PASS': return 'badge-pass';
    case 'FAIL': return 'badge-fail';
    case 'MANUAL': return 'badge-manual';
    default: return 'badge-skip';
  }
}

function severityBadgeClass(severity: Severity): string {
  switch (severity) {
    case 'high': return 'badge-high';
    case 'medium': return 'badge-medium';
    case 'low': return 'badge-low';
    default: return 'badge-skip';
  }
}

export default function CheckDetailPage() {
  const { name } = useParams<{ name: string }>();
  const [detail, setDetail] = useState<CheckResultDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!name) return;
    let cancelled = false;
    resultsApi.getDetail(name)
      .then(data => { if (!cancelled) setDetail(data); })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to fetch check detail'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [name]);

  if (loading) {
    return (
      <div className="card p-8 text-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 mx-auto mb-3" />
        <p className="text-sm text-gray-500">Loading check detail...</p>
      </div>
    );
  }

  if (error || !detail) {
    return (
      <div className="space-y-4">
        <Link to="/results" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:text-primary-800">
          <ArrowLeft className="h-4 w-4" /> Back to Results
        </Link>
        <div className="card border-red-200 bg-red-50 p-6 text-center">
          <p className="text-sm text-red-700">{error || 'Check not found'}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to="/results" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:text-primary-800">
        <ArrowLeft className="h-4 w-4" /> Back to Results
      </Link>

      <div>
        <h1 className="text-2xl font-bold text-gray-900 font-mono break-all">{detail.name}</h1>
        <div className="flex flex-wrap items-center gap-2 mt-2">
          <span className={`badge ${statusBadgeClass(detail.status)}`}>{detail.status}</span>
          {detail.severity && (
            <span className={`badge ${severityBadgeClass(detail.severity)}`}>{detail.severity}</span>
          )}
        </div>
      </div>

      {/* Remediation call-to-action for FAIL items */}
      {detail.status === 'FAIL' && detail.has_remediation && detail.remediation_name && (
        <div className="card border-amber-200 bg-amber-50 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Wrench className="h-5 w-5 text-amber-600" />
              <div>
                <p className="text-sm font-medium text-amber-900">Remediation Available</p>
                <p className="text-xs text-amber-700">
                  An automated fix is available for this check.
                </p>
              </div>
            </div>
            <Link
              to={`/remediation/${encodeURIComponent(detail.remediation_name)}`}
              className="btn btn-primary text-xs px-4 py-2"
            >
              <Wrench className="h-3.5 w-3.5 mr-1" />
              View Remediation
            </Link>
          </div>
        </div>
      )}

      {detail.status === 'FAIL' && !detail.has_remediation && (
        <div className="card border-gray-200 bg-gray-50 p-4">
          <div className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-gray-400" />
            <div>
              <p className="text-sm font-medium text-gray-700">No Automated Remediation</p>
              <p className="text-xs text-gray-500">
                This check requires manual remediation. See the instructions below.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Description */}
      <div className="card">
        <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
          <h2 className="font-medium text-sm text-gray-900">Description</h2>
        </div>
        <div className="p-4">
          <p className="text-sm text-gray-700 whitespace-pre-wrap">{detail.description || 'No description available.'}</p>
        </div>
      </div>

      {/* Rationale */}
      {detail.rationale && (
        <div className="card">
          <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
            <h2 className="font-medium text-sm text-gray-900">Rationale</h2>
          </div>
          <div className="p-4">
            <p className="text-sm text-gray-700 whitespace-pre-wrap">{detail.rationale}</p>
          </div>
        </div>
      )}

      {/* Instructions */}
      {detail.instructions && (
        <div className="card">
          <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
            <h2 className="font-medium text-sm text-gray-900">Instructions</h2>
          </div>
          <div className="p-4">
            <pre className="text-sm text-gray-700 whitespace-pre-wrap font-mono bg-gray-50 rounded-lg p-4">{detail.instructions}</pre>
          </div>
        </div>
      )}

      {/* Metadata */}
      <div className="card">
        <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
          <h2 className="font-medium text-sm text-gray-900">Metadata</h2>
        </div>
        <div className="p-4 grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-gray-500">Rule ID</span>
            <p className="font-mono text-xs text-gray-900 break-all">{detail.id || 'N/A'}</p>
          </div>
          <div>
            <span className="text-gray-500">Severity</span>
            <p className="font-medium text-gray-900 capitalize">{detail.severity || 'N/A'}</p>
          </div>
          <div>
            <span className="text-gray-500">Status</span>
            <p className="font-medium text-gray-900">{detail.status}</p>
          </div>
          {detail.scan_name && (
            <div>
              <span className="text-gray-500">Scan</span>
              <p className="font-medium text-gray-900">
                <span className="badge bg-blue-100 text-blue-700">{detail.scan_name}</span>
              </p>
            </div>
          )}
          {detail.suite && (
            <div>
              <span className="text-gray-500">Suite</span>
              <p className="font-medium text-gray-900">{detail.suite}</p>
            </div>
          )}
          {detail.has_remediation && (
            <div>
              <span className="text-gray-500">Remediation</span>
              <p>
                <Link
                  to={`/remediation/${encodeURIComponent(detail.remediation_name!)}`}
                  className="text-primary-600 hover:text-primary-800 hover:underline font-mono text-xs"
                >
                  {detail.remediation_name}
                </Link>
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
