import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Shield, RotateCw, Clock } from 'lucide-react';
import { remediationApi } from '../lib/api';
import type { RemediationDetail, Severity } from '../types/api';

function severityBadgeClass(severity: Severity): string {
  switch (severity) {
    case 'high': return 'badge-high';
    case 'medium': return 'badge-medium';
    case 'low': return 'badge-low';
    default: return 'badge-skip';
  }
}

export default function RemediationDetailPage() {
  const { name } = useParams<{ name: string }>();
  const [detail, setDetail] = useState<RemediationDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!name) return;
    setLoading(true);
    remediationApi.getDetail(name)
      .then(setDetail)
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to fetch detail'))
      .finally(() => setLoading(false));
  }, [name]);

  // Read applied timestamp from localStorage
  const appliedAt = name ? localStorage.getItem(`remediation-applied-${name}`) : null;

  if (loading) {
    return (
      <div className="card p-8 text-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 mx-auto mb-3" />
        <p className="text-sm text-gray-500">Loading remediation detail...</p>
      </div>
    );
  }

  if (error || !detail) {
    return (
      <div className="space-y-4">
        <Link to="/remediation" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:text-primary-800">
          <ArrowLeft className="h-4 w-4" /> Back to Remediations
        </Link>
        <div className="card border-red-200 bg-red-50 p-6 text-center">
          <p className="text-sm text-red-700">{error || 'Remediation not found'}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to="/remediation" className="inline-flex items-center gap-1 text-sm text-primary-600 hover:text-primary-800">
        <ArrowLeft className="h-4 w-4" /> Back to Remediations
      </Link>

      <div>
        <h1 className="text-2xl font-bold text-gray-900 font-mono break-all">{detail.name}</h1>
        <div className="flex flex-wrap items-center gap-2 mt-2">
          {detail.severity && (
            <span className={`badge ${severityBadgeClass(detail.severity)}`}>{detail.severity}</span>
          )}
          {detail.kind && (
            <span className="badge bg-gray-100 text-gray-600">{detail.kind}</span>
          )}
          {detail.applied && (
            <span className="badge bg-emerald-100 text-emerald-700">Applied</span>
          )}
          {detail.reboot_needed && (
            <span className="inline-flex items-center gap-1 badge bg-amber-100 text-amber-700">
              <RotateCw className="h-3 w-3" /> Reboot Required
            </span>
          )}
        </div>
      </div>

      {/* Metadata card */}
      <div className="card">
        <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
          <h2 className="font-medium text-sm text-gray-900">Details</h2>
        </div>
        <div className="p-4 grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-gray-500">Kind</span>
            <p className="font-medium text-gray-900">{detail.kind || 'N/A'}</p>
          </div>
          <div>
            <span className="text-gray-500">API Version</span>
            <p className="font-medium text-gray-900">{detail.api_version || 'N/A'}</p>
          </div>
          <div>
            <span className="text-gray-500">Target Namespace</span>
            <p className="font-medium text-gray-900">{detail.namespace || 'cluster-scoped'}</p>
          </div>
          <div>
            <span className="text-gray-500">Role</span>
            <p className="font-medium text-gray-900">{detail.role || 'N/A'}</p>
          </div>
          <div>
            <span className="text-gray-500">Status</span>
            <p className="font-medium text-gray-900">
              {detail.applied ? (
                <span className="inline-flex items-center gap-1 text-emerald-700">
                  <Shield className="h-4 w-4" /> Applied
                </span>
              ) : 'Pending'}
            </p>
          </div>
          {appliedAt && (
            <div>
              <span className="text-gray-500">Applied At</span>
              <p className="font-medium text-gray-900 inline-flex items-center gap-1">
                <Clock className="h-4 w-4 text-gray-400" />
                {new Date(appliedAt).toLocaleString()}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* YAML card */}
      <div className="card">
        <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
          <h2 className="font-medium text-sm text-gray-900">Remediation Object YAML</h2>
        </div>
        <div className="p-4">
          {detail.object_yaml ? (
            <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 overflow-x-auto text-xs font-mono leading-relaxed whitespace-pre">
              {detail.object_yaml}
            </pre>
          ) : (
            <p className="text-sm text-gray-500">No object YAML available for this remediation.</p>
          )}
        </div>
      </div>
    </div>
  );
}
