import { useEffect, useState, useCallback } from 'react';
import { Radar, Clock, CheckCircle2, XCircle, AlertTriangle, Plus, Play } from 'lucide-react';
import { scanApi } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import type { SuiteStatus, ScanStatus, ProfileInfo } from '../types/api';

function phaseIcon(phase: string) {
  switch (phase) {
    case 'DONE':
      return <CheckCircle2 className="h-5 w-5 text-emerald-500" />;
    case 'RUNNING':
    case 'AGGREGATING':
    case 'LAUNCHING':
      return (
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary-300 border-t-primary-600" />
      );
    case 'ERROR':
      return <XCircle className="h-5 w-5 text-red-500" />;
    default:
      return <Clock className="h-5 w-5 text-gray-400" />;
  }
}

function resultBadge(result: string | undefined) {
  if (!result) return null;
  switch (result) {
    case 'COMPLIANT':
      return <span className="badge bg-emerald-100 text-emerald-700">Compliant</span>;
    case 'NON-COMPLIANT':
      return <span className="badge bg-red-100 text-red-700">Non-Compliant</span>;
    case 'INCONSISTENT':
      return <span className="badge bg-amber-100 text-amber-700">Inconsistent</span>;
    case 'ERROR':
      return <span className="badge bg-red-100 text-red-700">Error</span>;
    default:
      return <span className="badge bg-gray-100 text-gray-600">{result}</span>;
  }
}

function phaseBadge(phase: string) {
  switch (phase) {
    case 'DONE':
      return <span className="badge bg-emerald-100 text-emerald-700">Done</span>;
    case 'RUNNING':
      return <span className="badge bg-blue-100 text-blue-700">Running</span>;
    case 'AGGREGATING':
      return <span className="badge bg-blue-100 text-blue-700">Aggregating</span>;
    case 'LAUNCHING':
      return <span className="badge bg-blue-100 text-blue-700">Launching</span>;
    case 'ERROR':
      return <span className="badge bg-red-100 text-red-700">Error</span>;
    case 'PENDING':
      return <span className="badge bg-gray-100 text-gray-600">Pending</span>;
    default:
      return <span className="badge bg-gray-100 text-gray-600">{phase || 'Unknown'}</span>;
  }
}

function formatDuration(start?: string, end?: string): string | null {
  if (!start || !end) return null;
  const startDate = new Date(start);
  const endDate = new Date(end);
  const diffMs = endDate.getTime() - startDate.getTime();
  if (diffMs < 0) return null;
  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSec = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSec}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMin = minutes % 60;
  return `${hours}h ${remainingMin}m`;
}

function ScanCard({ scan }: { scan: ScanStatus }) {
  const duration = formatDuration(scan.start_timestamp, scan.end_timestamp);

  return (
    <div className="border border-gray-200 rounded-lg p-4 bg-white">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          {phaseIcon(scan.phase)}
          <h4 className="font-mono text-sm font-medium text-gray-900">{scan.name}</h4>
        </div>
        <div className="flex items-center gap-2">
          {phaseBadge(scan.phase)}
          {resultBadge(scan.result)}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 text-xs">
        {scan.profile && (
          <div>
            <span className="text-gray-500 block">Profile</span>
            <span className="text-gray-900 font-mono break-all">{scan.profile}</span>
          </div>
        )}
        {scan.scan_type && (
          <div>
            <span className="text-gray-500 block">Type</span>
            <span className="text-gray-900">{scan.scan_type}</span>
          </div>
        )}
        {scan.start_timestamp && (
          <div>
            <span className="text-gray-500 block">Started</span>
            <span className="text-gray-900">{new Date(scan.start_timestamp).toLocaleString()}</span>
          </div>
        )}
        {scan.end_timestamp && (
          <div>
            <span className="text-gray-500 block">Completed</span>
            <span className="text-gray-900">{new Date(scan.end_timestamp).toLocaleString()}</span>
          </div>
        )}
        {duration && (
          <div>
            <span className="text-gray-500 block">Duration</span>
            <span className="text-gray-900">{duration}</span>
          </div>
        )}
        {scan.content_image && (
          <div className="col-span-2">
            <span className="text-gray-500 block">Content Image</span>
            <span className="text-gray-900 font-mono break-all text-[11px]">{scan.content_image}</span>
          </div>
        )}
      </div>

      {scan.warnings && (
        <div className="mt-3 text-xs bg-amber-50 border border-amber-200 rounded-md p-2">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-3.5 w-3.5 text-amber-500 mt-0.5 shrink-0" />
            <div>
              <span className="font-medium text-amber-800">Scan Warning</span>
              <p className="text-amber-700 mt-0.5">{scan.warnings}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function ScansPage() {
  const [suites, setSuites] = useState<SuiteStatus[]>([]);
  const [profiles, setProfiles] = useState<ProfileInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showNewScan, setShowNewScan] = useState(false);
  const [selectedProfile, setSelectedProfile] = useState('');
  const [scanName, setScanName] = useState('');
  const [creating, setCreating] = useState(false);
  const [createMsg, setCreateMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const { updateCounter, clusterStatus } = useDashboardStore();

  const fetchScans = useCallback(async () => {
    try {
      const data = await scanApi.list();
      setSuites(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch scans');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchProfiles = useCallback(async () => {
    try {
      const data = await scanApi.listProfiles();
      setProfiles(data || []);
    } catch {
      // Non-fatal
    }
  }, []);

  useEffect(() => {
    if (clusterStatus?.connected) {
      const timer = setTimeout(() => {
        fetchScans();
        fetchProfiles();
      }, updateCounter === 0 ? 0 : 2000);
      return () => clearTimeout(timer);
    }
  }, [clusterStatus?.connected, updateCounter, fetchScans, fetchProfiles]);

  // Auto-generate scan name from profile
  useEffect(() => {
    if (selectedProfile) {
      setScanName(selectedProfile + '-scan');
    }
  }, [selectedProfile]);

  const handleCreateScan = async () => {
    if (!selectedProfile) return;
    setCreating(true);
    setCreateMsg(null);
    try {
      await scanApi.create({
        name: scanName || selectedProfile + '-scan',
        profile: selectedProfile,
      });
      setCreateMsg({ type: 'success', text: `Scan "${scanName}" created. It will appear below once it starts.` });
      setShowNewScan(false);
      setSelectedProfile('');
      setScanName('');
      // Refresh after a short delay to let the scan register
      setTimeout(fetchScans, 3000);
    } catch (err) {
      setCreateMsg({ type: 'error', text: err instanceof Error ? err.message : 'Failed to create scan' });
    } finally {
      setCreating(false);
    }
  };

  // Determine which profiles already have a scan running or done
  const scannedProfiles = new Set<string>();
  for (const suite of suites) {
    for (const scan of suite.scans || []) {
      // The scan name is typically the profile name
      scannedProfiles.add(scan.name);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Scans</h1>
          <p className="text-sm text-gray-500 mt-1">
            Compliance scan suites and their individual scans
          </p>
        </div>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={fetchScans}>
            Refresh
          </button>
          <button
            className="btn btn-primary"
            onClick={() => setShowNewScan(!showNewScan)}
          >
            <Plus className="h-4 w-4 mr-1" />
            New Scan
          </button>
        </div>
      </div>

      {/* Create scan messages */}
      {createMsg && (
        <div className={`card p-4 ${createMsg.type === 'success' ? 'border-emerald-200 bg-emerald-50' : 'border-red-200 bg-red-50'}`}>
          <p className={`text-sm ${createMsg.type === 'success' ? 'text-emerald-700' : 'text-red-700'}`}>
            {createMsg.text}
          </p>
        </div>
      )}

      {/* New Scan panel */}
      {showNewScan && (
        <div className="card">
          <div className="px-5 py-4 border-b border-gray-200 bg-gray-50">
            <h2 className="font-semibold text-gray-900">Create New Scan</h2>
            <p className="text-xs text-gray-500 mt-0.5">
              Select a compliance profile to scan against
            </p>
          </div>
          <div className="p-5 space-y-4">
            {profiles.length === 0 ? (
              <p className="text-sm text-gray-500">Loading profiles...</p>
            ) : (
              <>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Profile</label>
                  <select
                    className="input w-full"
                    value={selectedProfile}
                    onChange={e => setSelectedProfile(e.target.value)}
                  >
                    <option value="">Select a profile...</option>
                    {profiles.map(p => (
                      <option key={p.name} value={p.name}>
                        {p.name} — {p.title}
                      </option>
                    ))}
                  </select>
                </div>
                {selectedProfile && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Scan Name</label>
                      <input
                        type="text"
                        className="input w-full"
                        value={scanName}
                        onChange={e => setScanName(e.target.value)}
                        placeholder="e.g. my-cis-scan"
                      />
                    </div>
                    {profiles.find(p => p.name === selectedProfile)?.description && (
                      <div className="bg-gray-50 rounded-lg p-3 text-xs text-gray-600">
                        {profiles.find(p => p.name === selectedProfile)?.description}
                      </div>
                    )}
                  </>
                )}
                <div className="flex gap-2">
                  <button
                    className="btn btn-primary"
                    disabled={!selectedProfile || creating}
                    onClick={handleCreateScan}
                  >
                    {creating ? (
                      <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent mr-1" />
                    ) : (
                      <Play className="h-4 w-4 mr-1" />
                    )}
                    {creating ? 'Creating...' : 'Start Scan'}
                  </button>
                  <button className="btn btn-secondary" onClick={() => setShowNewScan(false)}>
                    Cancel
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Existing scans */}
      {loading ? (
        <div className="card p-8 text-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600 mx-auto mb-3" />
          <p className="text-sm text-gray-500">Loading scans...</p>
        </div>
      ) : error ? (
        <div className="card border-red-200 bg-red-50 p-6 text-center">
          <p className="text-sm text-red-700">{error}</p>
          <button className="btn btn-secondary mt-3" onClick={fetchScans}>
            Retry
          </button>
        </div>
      ) : suites.length === 0 ? (
        <div className="card p-12 text-center">
          <Radar className="h-12 w-12 text-gray-300 mx-auto mb-3" />
          <p className="text-gray-500">No compliance scans found.</p>
          <p className="text-sm text-gray-400 mt-1">
            Click "New Scan" above to create your first scan.
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {suites.map(suite => (
            <div key={suite.name} className="card">
              {/* Suite header */}
              <div className="px-5 py-4 border-b border-gray-200 bg-gray-50">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    {phaseIcon(suite.phase)}
                    <div>
                      <h2 className="font-semibold text-gray-900">{suite.name}</h2>
                      {suite.created_at && (
                        <p className="text-xs text-gray-500 mt-0.5">
                          Created {new Date(suite.created_at).toLocaleString()}
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {phaseBadge(suite.phase)}
                    {resultBadge(suite.result)}
                  </div>
                </div>
              </div>

              {/* Conditions */}
              {suite.conditions && suite.conditions.length > 0 && (
                <div className="px-5 py-3 border-b border-gray-100">
                  <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Conditions</h3>
                  <div className="flex flex-wrap gap-3">
                    {suite.conditions.map(cond => (
                      <div key={cond.type} className="flex items-center gap-1.5 text-xs">
                        {cond.status === 'True' ? (
                          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
                        ) : (
                          <XCircle className="h-3.5 w-3.5 text-gray-400" />
                        )}
                        <span className="text-gray-700 font-medium">{cond.type}</span>
                        {cond.reason && (
                          <span className="text-gray-500">({cond.reason})</span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Scans */}
              <div className="p-5">
                <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
                  Scans ({suite.scans?.length || 0})
                </h3>
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                  {suite.scans?.map(scan => (
                    <ScanCard key={scan.name} scan={scan} />
                  ))}
                </div>
                {(!suite.scans || suite.scans.length === 0) && (
                  <p className="text-sm text-gray-500">No scans in this suite.</p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Available Profiles */}
      {!loading && profiles.length > 0 && (
        <div className="card">
          <div className="px-5 py-4 border-b border-gray-200 bg-gray-50">
            <h2 className="font-semibold text-gray-900">Available Profiles</h2>
            <p className="text-xs text-gray-500 mt-0.5">
              {profiles.length} compliance profiles installed
            </p>
          </div>
          <div className="divide-y divide-gray-100">
            {profiles.map(profile => {
              const hasBeenScanned = scannedProfiles.has(profile.name);
              return (
                <div key={profile.name} className="px-5 py-3 flex items-center justify-between">
                  <div className="flex-1 min-w-0 mr-4">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs text-gray-900">{profile.name}</span>
                      {hasBeenScanned && (
                        <span className="badge bg-emerald-100 text-emerald-700 text-[10px]">Scanned</span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 mt-0.5 truncate">{profile.title}</p>
                  </div>
                  <button
                    className="btn btn-secondary text-xs px-3 py-1.5"
                    onClick={() => {
                      setSelectedProfile(profile.name);
                      setShowNewScan(true);
                      window.scrollTo({ top: 0, behavior: 'smooth' });
                    }}
                  >
                    <Play className="h-3 w-3 mr-1" />
                    Scan
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
