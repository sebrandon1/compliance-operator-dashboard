import { useEffect, useState, useCallback, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Shield, Clock, Search } from 'lucide-react';
import RemediationPanel from '../components/RemediationPanel';
import { remediationApi } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import type { RemediationInfo, Severity } from '../types/api';

type SortField = 'severity' | 'name' | 'reboot';
type SortDirection = 'asc' | 'desc';

const severityOrder: Record<string, number> = { high: 0, medium: 1, low: 2 };

export default function RemediationPage() {
  const [remediations, setRemediations] = useState<RemediationInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { updateCounter, clusterStatus } = useDashboardStore();

  // Filter & sort state
  const [search, setSearch] = useState('');
  const [severityFilter, setSeverityFilter] = useState<Severity | ''>('');
  const [rebootFilter, setRebootFilter] = useState<'' | 'yes' | 'no'>('');
  const [sortField, setSortField] = useState<SortField>('severity');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

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
      const timer = setTimeout(() => {
        fetchRemediations();
      }, updateCounter === 0 ? 0 : 2000);
      return () => clearTimeout(timer);
    }
  }, [clusterStatus?.connected, updateCounter, fetchRemediations]);

  // Applied MachineConfigs needing reboot stay in the pending panel so users
  // can see "Requires Reboot" and remove them before the MCO reboots nodes.
  const notApplied = remediations.filter(r => !r.applied || (r.applied && r.reboot_needed));
  const applied = remediations.filter(r => r.applied && !r.reboot_needed);

  // Apply filters & sort
  const filtered = useMemo(() => {
    let items = notApplied;

    if (search) {
      const lower = search.toLowerCase();
      items = items.filter(r =>
        r.name.toLowerCase().includes(lower) ||
        (r.role && r.role.toLowerCase().includes(lower)) ||
        (r.kind && r.kind.toLowerCase().includes(lower))
      );
    }

    if (severityFilter) {
      items = items.filter(r => r.severity === severityFilter);
    }

    if (rebootFilter === 'yes') {
      items = items.filter(r => r.reboot_needed);
    } else if (rebootFilter === 'no') {
      items = items.filter(r => !r.reboot_needed);
    }

    const dir = sortDirection === 'asc' ? 1 : -1;
    items = [...items].sort((a, b) => {
      switch (sortField) {
        case 'severity':
          return ((severityOrder[a.severity] ?? 3) - (severityOrder[b.severity] ?? 3)) * dir;
        case 'name':
          return a.name.localeCompare(b.name) * dir;
        case 'reboot': {
          const aVal = a.reboot_needed ? 0 : 1;
          const bVal = b.reboot_needed ? 0 : 1;
          return (aVal - bVal) * dir;
        }
        default:
          return 0;
      }
    });

    return items;
  }, [notApplied, search, severityFilter, rebootFilter, sortField, sortDirection]);

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

              {/* Filters — matches ResultsTable pattern */}
              <div className="flex flex-wrap gap-3 mb-4">
                <div className="relative flex-1 min-w-[200px]">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    type="text"
                    placeholder="Search remediations..."
                    className="input pl-9"
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                  />
                </div>
                <select
                  className="input w-auto"
                  value={severityFilter}
                  onChange={e => setSeverityFilter(e.target.value as Severity | '')}
                >
                  <option value="">All Severities</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
                <select
                  className="input w-auto"
                  value={rebootFilter}
                  onChange={e => setRebootFilter(e.target.value as '' | 'yes' | 'no')}
                >
                  <option value="">All Reboot Status</option>
                  <option value="yes">Requires Reboot</option>
                  <option value="no">No Reboot</option>
                </select>
                <select
                  className="input w-auto"
                  value={`${sortField}-${sortDirection}`}
                  onChange={e => {
                    const [f, d] = e.target.value.split('-') as [SortField, SortDirection];
                    setSortField(f);
                    setSortDirection(d);
                  }}
                >
                  <option value="severity-asc">Sort: Severity (High first)</option>
                  <option value="severity-desc">Sort: Severity (Low first)</option>
                  <option value="name-asc">Sort: Name (A-Z)</option>
                  <option value="name-desc">Sort: Name (Z-A)</option>
                  <option value="reboot-asc">Sort: Reboot Required first</option>
                  <option value="reboot-desc">Sort: No Reboot first</option>
                </select>
              </div>

              <p className="text-sm text-gray-500 mb-3">
                Showing {filtered.length} of {notApplied.length} remediations
              </p>

              {filtered.length > 0 ? (
                <RemediationPanel remediations={filtered} onApplied={fetchRemediations} />
              ) : (
                <div className="card p-6 text-center text-gray-500 text-sm">
                  No remediations match your filters
                </div>
              )}
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
