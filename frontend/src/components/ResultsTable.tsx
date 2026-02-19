import { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Search, ChevronUp, ChevronDown, Wrench } from 'lucide-react';
import type { CheckResult, Severity, CheckStatus } from '../types/api';

interface ResultsTableProps {
  results: CheckResult[];
  initialSeverity?: string;
  remediationNames?: Set<string>;
}

type SortField = 'name' | 'severity' | 'status' | 'scan';
type SortDirection = 'asc' | 'desc';

const severityOrder: Record<string, number> = { high: 0, medium: 1, low: 2 };
const statusOrder: Record<string, number> = { FAIL: 0, MANUAL: 1, PASS: 2, SKIP: 3, 'NOT-APPLICABLE': 4 };

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

// Find a matching remediation name for a check (exact or prefix match)
function findRemediation(checkName: string, remediationNames?: Set<string>): string | null {
  if (!remediationNames) return null;
  if (remediationNames.has(checkName)) return checkName;
  for (const remName of remediationNames) {
    if (remName.startsWith(checkName + '-')) return remName;
  }
  return null;
}

export default function ResultsTable({ results, initialSeverity = '', remediationNames }: ResultsTableProps) {
  const [search, setSearch] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>(initialSeverity);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [scanFilter, setScanFilter] = useState<string>('');
  const [sortField, setSortField] = useState<SortField>('severity');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  // Collect unique scan names for the filter dropdown
  const scanNames = useMemo(() => {
    const names = new Set<string>();
    for (const r of results) {
      if (r.scan_name) names.add(r.scan_name);
    }
    return Array.from(names).sort();
  }, [results]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const filtered = useMemo(() => {
    let items = [...results];

    if (search) {
      const lower = search.toLowerCase();
      items = items.filter(r =>
        r.name.toLowerCase().includes(lower) ||
        r.description.toLowerCase().includes(lower)
      );
    }

    if (severityFilter) {
      items = items.filter(r => r.severity === severityFilter);
    }

    if (statusFilter) {
      items = items.filter(r => r.status === statusFilter);
    }

    if (scanFilter) {
      items = items.filter(r => r.scan_name === scanFilter);
    }

    items.sort((a, b) => {
      let cmp = 0;
      switch (sortField) {
        case 'name':
          cmp = a.name.localeCompare(b.name);
          break;
        case 'severity':
          cmp = (severityOrder[a.severity] ?? 99) - (severityOrder[b.severity] ?? 99);
          break;
        case 'status':
          cmp = (statusOrder[a.status] ?? 99) - (statusOrder[b.status] ?? 99);
          break;
        case 'scan':
          cmp = (a.scan_name || '').localeCompare(b.scan_name || '');
          break;
      }
      return sortDirection === 'asc' ? cmp : -cmp;
    });

    return items;
  }, [results, search, severityFilter, statusFilter, scanFilter, sortField, sortDirection]);

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return null;
    return sortDirection === 'asc'
      ? <ChevronUp className="h-4 w-4 inline" />
      : <ChevronDown className="h-4 w-4 inline" />;
  };

  return (
    <div>
      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-4">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search checks..."
            className="input pl-9"
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
        <select
          className="input w-auto"
          value={severityFilter}
          onChange={e => setSeverityFilter(e.target.value)}
        >
          <option value="">All Severities</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
        <select
          className="input w-auto"
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value)}
        >
          <option value="">All Statuses</option>
          <option value="PASS">Pass</option>
          <option value="FAIL">Fail</option>
          <option value="MANUAL">Manual</option>
          <option value="SKIP">Skip</option>
        </select>
        {scanNames.length > 0 && (
          <select
            className="input w-auto"
            value={scanFilter}
            onChange={e => setScanFilter(e.target.value)}
          >
            <option value="">All Scans</option>
            {scanNames.map(name => (
              <option key={name} value={name}>{name}</option>
            ))}
          </select>
        )}
      </div>

      {/* Results count */}
      <p className="text-sm text-gray-500 mb-3">
        Showing {filtered.length} of {results.length} results
      </p>

      {/* Table */}
      <div className="card overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th
                className="text-left px-4 py-3 font-medium text-gray-700 cursor-pointer hover:text-gray-900"
                onClick={() => handleSort('name')}
              >
                Check Name <SortIcon field="name" />
              </th>
              <th
                className="text-left px-4 py-3 font-medium text-gray-700 cursor-pointer hover:text-gray-900 w-28"
                onClick={() => handleSort('scan')}
              >
                Scan <SortIcon field="scan" />
              </th>
              <th
                className="text-left px-4 py-3 font-medium text-gray-700 cursor-pointer hover:text-gray-900 w-28"
                onClick={() => handleSort('severity')}
              >
                Severity <SortIcon field="severity" />
              </th>
              <th
                className="text-left px-4 py-3 font-medium text-gray-700 cursor-pointer hover:text-gray-900 w-28"
                onClick={() => handleSort('status')}
              >
                Status <SortIcon field="status" />
              </th>
              <th className="text-left px-4 py-3 font-medium text-gray-700">
                Description
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {filtered.map((result) => {
              const remName = result.status === 'FAIL' ? findRemediation(result.name, remediationNames) : null;
              return (
                <tr key={result.name} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <Link
                      to={`/results/${encodeURIComponent(result.name)}`}
                      className="font-mono text-xs text-primary-600 hover:text-primary-800 hover:underline"
                    >
                      {result.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3">
                    {result.scan_name && (
                      <span className="badge bg-blue-100 text-blue-700 text-[10px]">
                        {result.scan_name}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`badge ${severityBadgeClass(result.severity)}`}>
                      {result.severity}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className={`badge ${statusBadgeClass(result.status)}`}>
                        {result.status}
                      </span>
                      {remName && (
                        <Link
                          to={`/remediation/${encodeURIComponent(remName)}`}
                          className="inline-flex items-center gap-1 text-xs text-amber-700 hover:text-amber-900 hover:underline"
                          title="View remediation"
                        >
                          <Wrench className="h-3 w-3" />
                          Fix
                        </Link>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-gray-600 max-w-md truncate">
                    {result.description}
                  </td>
                </tr>
              );
            })}
            {filtered.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                  No results match your filters
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
