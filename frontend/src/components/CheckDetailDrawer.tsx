import { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { X, ExternalLink, Shield, Wrench } from 'lucide-react';
import { resultsApi } from '../lib/api';
import type { CheckResultDetail, CheckStatus, Severity } from '../types/api';

interface CheckDetailDrawerProps {
  checkName: string | null;
  onClose: () => void;
}

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

export default function CheckDetailDrawer({ checkName, onClose }: CheckDetailDrawerProps) {
  const [detail, setDetail] = useState<CheckResultDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [fetchedName, setFetchedName] = useState<string | null>(null);
  const [animateIn, setAnimateIn] = useState(false);

  // Derive loading state: we have a checkName but haven't finished fetching it yet
  const loading = checkName !== null && checkName !== fetchedName;

  // Trigger slide-in animation after mount/render via rAF
  useEffect(() => {
    if (!checkName) return;
    const id = requestAnimationFrame(() => setAnimateIn(true));
    return () => {
      cancelAnimationFrame(id);
      setAnimateIn(false);
    };
  }, [checkName]);

  // Fetch detail when checkName changes
  useEffect(() => {
    if (!checkName) return;

    let cancelled = false;
    resultsApi.getDetail(checkName)
      .then(data => {
        if (!cancelled) {
          setDetail(data);
          setError(null);
          setFetchedName(checkName);
        }
      })
      .catch(err => {
        if (!cancelled) {
          setDetail(null);
          setError(err instanceof Error ? err.message : 'Failed to fetch check detail');
          setFetchedName(checkName);
        }
      });

    return () => { cancelled = true; };
  }, [checkName]);

  // Close on Escape key
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
  }, [onClose]);

  useEffect(() => {
    if (checkName) {
      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [checkName, handleKeyDown]);

  if (!checkName) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className={`fixed inset-0 bg-black/30 z-40 transition-opacity duration-300 ${animateIn ? 'opacity-100' : 'opacity-0'}`}
        onClick={onClose}
      />

      {/* Drawer panel */}
      <div
        className={`fixed inset-y-0 right-0 z-50 w-full max-w-lg bg-white shadow-xl overflow-y-auto transition-transform duration-300 ${animateIn ? 'translate-x-0' : 'translate-x-full'}`}
      >
        {/* Header */}
        <div className="sticky top-0 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between z-10">
          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="p-1 rounded hover:bg-gray-100 text-gray-500 hover:text-gray-700"
            >
              <X className="h-5 w-5" />
            </button>
            <span className="text-sm font-medium text-gray-700">Check Detail</span>
          </div>
          <Link
            to={`/results/${encodeURIComponent(checkName)}`}
            className="inline-flex items-center gap-1 text-sm text-primary-600 hover:text-primary-800"
            onClick={(e) => e.stopPropagation()}
          >
            Open full page <ExternalLink className="h-3.5 w-3.5" />
          </Link>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {loading && (
            <div className="py-12 text-center">
              <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 mx-auto mb-3" />
              <p className="text-sm text-gray-500">Loading check detail...</p>
            </div>
          )}

          {!loading && error && (
            <div className="card border-red-200 bg-red-50 p-4 text-center">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {!loading && detail && (
            <>
              {/* Name + badges */}
              <div>
                <h2 className="text-lg font-bold text-gray-900 font-mono break-all">{detail.name}</h2>
                <div className="flex flex-wrap items-center gap-2 mt-2">
                  <span className={`badge ${statusBadgeClass(detail.status)}`}>{detail.status}</span>
                  {detail.severity && (
                    <span className={`badge ${severityBadgeClass(detail.severity)}`}>{detail.severity}</span>
                  )}
                </div>
              </div>

              {/* Remediation CTA */}
              {detail.status === 'FAIL' && detail.has_remediation && detail.remediation_name && (
                <div className="card border-amber-200 bg-amber-50 p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Wrench className="h-5 w-5 text-amber-600" />
                      <div>
                        <p className="text-sm font-medium text-amber-900">Remediation Available</p>
                        <p className="text-xs text-amber-700">An automated fix is available for this check.</p>
                      </div>
                    </div>
                    <Link
                      to={`/remediation/${encodeURIComponent(detail.remediation_name)}`}
                      className="btn btn-primary text-xs px-3 py-1.5"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <Wrench className="h-3.5 w-3.5 mr-1" />
                      View Fix
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
                      <p className="text-xs text-gray-500">This check requires manual remediation. See instructions below.</p>
                    </div>
                  </div>
                </div>
              )}

              {/* Description */}
              <div className="card">
                <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
                  <h3 className="font-medium text-sm text-gray-900">Description</h3>
                </div>
                <div className="p-4">
                  <p className="text-sm text-gray-700 whitespace-pre-wrap">{detail.description || 'No description available.'}</p>
                </div>
              </div>

              {/* Rationale */}
              {detail.rationale && (
                <div className="card">
                  <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
                    <h3 className="font-medium text-sm text-gray-900">Rationale</h3>
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
                    <h3 className="font-medium text-sm text-gray-900">Instructions</h3>
                  </div>
                  <div className="p-4">
                    <pre className="text-sm text-gray-700 whitespace-pre-wrap font-mono bg-gray-50 rounded-lg p-4">{detail.instructions}</pre>
                  </div>
                </div>
              )}

              {/* Metadata */}
              <div className="card">
                <div className="px-4 py-3 border-b border-gray-200 bg-gray-50">
                  <h3 className="font-medium text-sm text-gray-900">Metadata</h3>
                </div>
                <div className="p-4 grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-gray-500">Rule ID</span>
                    <p className="font-mono text-xs text-gray-900 break-all">{detail.id || 'N/A'}</p>
                  </div>
                  {detail.scan_name && (
                    <div>
                      <span className="text-gray-500">Scan</span>
                      <p><span className="badge bg-blue-100 text-blue-700">{detail.scan_name}</span></p>
                    </div>
                  )}
                  {detail.suite && (
                    <div>
                      <span className="text-gray-500">Suite</span>
                      <p className="font-medium text-gray-900">{detail.suite}</p>
                    </div>
                  )}
                  {detail.has_remediation && detail.remediation_name && (
                    <div>
                      <span className="text-gray-500">Remediation</span>
                      <p>
                        <Link
                          to={`/remediation/${encodeURIComponent(detail.remediation_name)}`}
                          className="text-primary-600 hover:text-primary-800 hover:underline font-mono text-xs"
                          onClick={(e) => e.stopPropagation()}
                        >
                          {detail.remediation_name}
                        </Link>
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </>
  );
}
