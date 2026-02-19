import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Play, AlertTriangle, RotateCw, Shield } from 'lucide-react';
import { remediationApi } from '../lib/api';
import type { RemediationInfo, Severity } from '../types/api';

interface RemediationPanelProps {
  remediations: RemediationInfo[];
  onApplied: () => void;
}

function severityBadgeClass(severity: Severity): string {
  switch (severity) {
    case 'high': return 'badge-high';
    case 'medium': return 'badge-medium';
    case 'low': return 'badge-low';
    default: return 'badge-skip';
  }
}

export default function RemediationPanel({ remediations, onApplied }: RemediationPanelProps) {
  const [applying, setApplying] = useState<string | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<RemediationInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const handleApply = async (rem: RemediationInfo) => {
    if (rem.reboot_needed) {
      setConfirmDialog(rem);
      return;
    }
    await doApply(rem.name);
  };

  const doApply = async (name: string) => {
    setConfirmDialog(null);
    setApplying(name);
    setError(null);
    setSuccessMsg(null);

    try {
      const result = await remediationApi.apply(name);
      if (result.applied) {
        localStorage.setItem(`remediation-applied-${name}`, new Date().toISOString());
        setSuccessMsg(result.message);
        onApplied();
      } else {
        setError(result.error || 'Failed to apply remediation');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply remediation');
    } finally {
      setApplying(null);
    }
  };

  // Group by severity
  const grouped: Record<Severity, RemediationInfo[]> = { high: [], medium: [], low: [] };
  for (const rem of remediations) {
    if (rem.severity in grouped) {
      grouped[rem.severity].push(rem);
    }
  }

  return (
    <div className="space-y-4">
      {error && (
        <div className="card border-red-200 bg-red-50 p-4">
          <p className="text-sm text-red-700">{error}</p>
        </div>
      )}
      {successMsg && (
        <div className="card border-emerald-200 bg-emerald-50 p-4">
          <p className="text-sm text-emerald-700">{successMsg}</p>
        </div>
      )}

      {(['high', 'medium', 'low'] as Severity[]).map(severity => {
        const items = grouped[severity];
        if (items.length === 0) return null;

        return (
          <div key={severity} className="card">
            <div className="px-4 py-3 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Shield className="h-4 w-4 text-gray-500" />
                <h3 className="font-medium text-sm text-gray-900 capitalize">
                  {severity} Severity
                </h3>
                <span className="badge bg-gray-200 text-gray-700">{items.length}</span>
              </div>
            </div>
            <div className="divide-y divide-gray-100">
              {items.map(rem => (
                <div key={rem.name} className="px-4 py-3 flex items-center justify-between hover:bg-gray-50">
                  <div className="flex-1 min-w-0 mr-4">
                    <div className="flex items-center gap-2">
                      <Link
                        to={`/remediation/${encodeURIComponent(rem.name)}`}
                        className="font-mono text-xs text-primary-600 hover:text-primary-800 hover:underline truncate"
                      >
                        {rem.name}
                      </Link>
                      <span className={`badge ${severityBadgeClass(rem.severity)}`}>{rem.severity}</span>
                      {rem.kind && (
                        <span className="badge bg-gray-100 text-gray-600">{rem.kind}</span>
                      )}
                      {rem.reboot_needed && (
                        <span title="Requires node reboot">
                          <RotateCw className="h-3.5 w-3.5 text-amber-500" />
                        </span>
                      )}
                      {rem.applied && (
                        <span className="badge bg-emerald-100 text-emerald-700">Applied</span>
                      )}
                    </div>
                    {rem.role && (
                      <span className="text-xs text-gray-500">Role: {rem.role}</span>
                    )}
                  </div>
                  <button
                    className="btn btn-primary text-xs px-3 py-1.5"
                    disabled={applying === rem.name || rem.applied}
                    onClick={() => handleApply(rem)}
                  >
                    {applying === rem.name ? (
                      <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                    ) : (
                      <>
                        <Play className="h-3.5 w-3.5 mr-1" />
                        Apply
                      </>
                    )}
                  </button>
                </div>
              ))}
            </div>
          </div>
        );
      })}

      {remediations.length === 0 && (
        <div className="card p-8 text-center text-gray-500">
          No remediations available
        </div>
      )}

      {/* Confirmation dialog for MachineConfig changes */}
      {confirmDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="h-6 w-6 text-amber-500" />
              <h3 className="font-semibold text-lg text-gray-900">Confirm Reboot</h3>
            </div>
            <p className="text-sm text-gray-600 mb-2">
              Applying <span className="font-mono font-medium">{confirmDialog.name}</span> will
              create a MachineConfig change that will trigger a node reboot for
              <span className="font-medium"> {confirmDialog.role || 'worker'}</span> nodes.
            </p>
            <p className="text-sm text-gray-600 mb-6">
              This operation cannot be easily undone. Proceed?
            </p>
            <div className="flex gap-3 justify-end">
              <button
                className="btn btn-secondary"
                onClick={() => setConfirmDialog(null)}
              >
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={() => doApply(confirmDialog.name)}
              >
                Apply & Reboot
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
