import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { Play, AlertTriangle, RotateCw, Shield, CheckSquare, Square, Loader2, Trash2 } from 'lucide-react';
import { remediationApi } from '../lib/api';
import type { RemediationInfo, RemediationResult, Severity } from '../types/api';

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

  const [removing, setRemoving] = useState<string | null>(null);

  // Batch selection state
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [batchApplying, setBatchApplying] = useState(false);
  const [batchProgress, setBatchProgress] = useState<{ current: number; total: number } | null>(null);
  const [, setBatchResults] = useState<{ succeeded: number; failed: number } | null>(null);
  const [showBatchConfirm, setShowBatchConfirm] = useState(false);

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

  const doRemove = async (name: string) => {
    setRemoving(name);
    setError(null);
    setSuccessMsg(null);

    try {
      const result = await remediationApi.remove(name);
      localStorage.removeItem(`remediation-applied-${name}`);
      setSuccessMsg(result.message);
      onApplied();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove remediation');
    } finally {
      setRemoving(null);
    }
  };

  // Selection helpers
  const toggleSelect = useCallback((name: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  const toggleSelectGroup = useCallback((items: RemediationInfo[]) => {
    const pending = items.filter(r => !r.applied);
    const allSelected = pending.every(r => selected.has(r.name));
    setSelected(prev => {
      const next = new Set(prev);
      for (const r of pending) {
        if (allSelected) {
          next.delete(r.name);
        } else {
          next.add(r.name);
        }
      }
      return next;
    });
  }, [selected]);

  const selectedHasReboot = Array.from(selected).some(name => {
    const rem = remediations.find(r => r.name === name);
    return rem?.reboot_needed;
  });

  // Batch apply
  const handleBatchApply = () => {
    if (selectedHasReboot) {
      setShowBatchConfirm(true);
      return;
    }
    doBatchApply();
  };

  const doBatchApply = async () => {
    setShowBatchConfirm(false);
    setError(null);
    setSuccessMsg(null);
    setBatchApplying(true);
    setBatchResults(null);

    const names = Array.from(selected);
    setBatchProgress({ current: 0, total: names.length });

    try {
      const results: RemediationResult[] = await remediationApi.applyBatch(names);

      let succeeded = 0;
      let failed = 0;
      for (const result of results) {
        if (result.applied) {
          succeeded++;
          localStorage.setItem(`remediation-applied-${result.name}`, new Date().toISOString());
        } else {
          failed++;
        }
      }

      setBatchProgress(null);
      setBatchResults({ succeeded, failed });
      setSelected(new Set());

      if (failed > 0 && succeeded === 0) {
        setError(`All ${failed} remediations failed to apply`);
      } else if (failed > 0) {
        setSuccessMsg(`Applied ${succeeded} of ${succeeded + failed} remediations (${failed} failed)`);
      } else {
        setSuccessMsg(`Successfully applied ${succeeded} remediation${succeeded !== 1 ? 's' : ''}`);
      }

      onApplied();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Batch apply failed');
      setBatchProgress(null);
    } finally {
      setBatchApplying(false);
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
    <div className="space-y-4 pb-20">
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

        const pendingInGroup = items.filter(r => !r.applied);
        const allGroupSelected = pendingInGroup.length > 0 && pendingInGroup.every(r => selected.has(r.name));

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
              {pendingInGroup.length > 0 && (
                <button
                  className="text-xs text-primary-600 hover:text-primary-800 font-medium flex items-center gap-1"
                  onClick={() => toggleSelectGroup(items)}
                  disabled={batchApplying}
                >
                  {allGroupSelected ? (
                    <><CheckSquare className="h-3.5 w-3.5" /> Deselect All</>
                  ) : (
                    <><Square className="h-3.5 w-3.5" /> Select All</>
                  )}
                </button>
              )}
            </div>
            <div className="divide-y divide-gray-100">
              {items.map(rem => (
                <div key={rem.name} className="px-4 py-3 flex items-center justify-between hover:bg-gray-50">
                  <div className="flex items-center flex-1 min-w-0 mr-4">
                    {!rem.applied && (
                      <button
                        className="mr-3 flex-shrink-0 text-gray-400 hover:text-primary-600"
                        onClick={() => toggleSelect(rem.name)}
                        disabled={batchApplying}
                      >
                        {selected.has(rem.name) ? (
                          <CheckSquare className="h-4 w-4 text-primary-600" />
                        ) : (
                          <Square className="h-4 w-4" />
                        )}
                      </button>
                    )}
                    <div className="flex-1 min-w-0">
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
                  </div>
                  {rem.applied && rem.reboot_needed ? (
                    <div className="flex items-center gap-2">
                      <span className="badge bg-amber-100 text-amber-700 flex items-center gap-1">
                        <RotateCw className="h-3 w-3" />
                        Requires Reboot
                      </span>
                      <button
                        className="btn btn-danger text-xs px-3 py-1.5"
                        disabled={removing === rem.name}
                        onClick={() => doRemove(rem.name)}
                      >
                        {removing === rem.name ? (
                          <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                        ) : (
                          <>
                            <Trash2 className="h-3.5 w-3.5 mr-1" />
                            Remove
                          </>
                        )}
                      </button>
                    </div>
                  ) : (
                    <button
                      className="btn btn-primary text-xs px-3 py-1.5"
                      disabled={applying === rem.name || rem.applied || batchApplying}
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
                  )}
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

      {/* Batch action bar */}
      {selected.size > 0 && (
        <div className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 shadow-lg z-40">
          <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <span className="text-sm font-medium text-gray-900">
                {selected.size} selected
              </span>
              {selectedHasReboot && (
                <span className="flex items-center gap-1 text-xs text-amber-600">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  Includes changes requiring node reboot
                </span>
              )}
            </div>
            <div className="flex items-center gap-3">
              <button
                className="btn btn-secondary text-xs"
                onClick={() => setSelected(new Set())}
                disabled={batchApplying}
              >
                Clear
              </button>
              <button
                className="btn btn-primary text-xs px-4"
                onClick={handleBatchApply}
                disabled={batchApplying}
              >
                {batchApplying ? (
                  <span className="flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Applying {batchProgress ? `${batchProgress.current} of ${batchProgress.total}` : '...'}
                  </span>
                ) : (
                  <>
                    <Play className="h-3.5 w-3.5 mr-1" />
                    Apply Selected
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Confirmation dialog for single MachineConfig changes */}
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

      {/* Batch confirmation dialog for reboot-required items */}
      {showBatchConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="h-6 w-6 text-amber-500" />
              <h3 className="font-semibold text-lg text-gray-900">Confirm Batch Apply</h3>
            </div>
            <p className="text-sm text-gray-600 mb-2">
              You are about to apply <span className="font-medium">{selected.size} remediations</span>.
              Some of these include MachineConfig changes that will trigger node reboots.
            </p>
            <p className="text-sm text-gray-600 mb-6">
              Applying them together consolidates changes before any reboot cycle.
              This operation cannot be easily undone. Proceed?
            </p>
            <div className="flex gap-3 justify-end">
              <button
                className="btn btn-secondary"
                onClick={() => setShowBatchConfirm(false)}
              >
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={doBatchApply}
              >
                Apply All & Reboot
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
