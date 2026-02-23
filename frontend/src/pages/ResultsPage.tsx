import { useMemo, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FileSearch } from 'lucide-react';
import ResultsTable from '../components/ResultsTable';
import CheckDetailDrawer from '../components/CheckDetailDrawer';
import { useCompliance } from '../hooks/useCompliance';
import { remediationApi } from '../lib/api';
import type { CheckResult } from '../types/api';

export default function ResultsPage() {
  const { complianceData, refresh } = useCompliance();
  const [searchParams] = useSearchParams();
  const initialSeverity = searchParams.get('severity') || '';
  const [remediationNames, setRemediationNames] = useState<Set<string>>(new Set());
  const [drawerCheckName, setDrawerCheckName] = useState<string | null>(null);

  // Fetch remediation names so we can show "Fix" links on FAIL items
  useEffect(() => {
    remediationApi.list()
      .then(rems => setRemediationNames(new Set((rems || []).map(r => r.name))))
      .catch(() => {}); // Ignore errors - just won't show Fix links
  }, []);

  const allResults = useMemo((): CheckResult[] => {
    if (!complianceData) return [];

    const results: CheckResult[] = [];

    // Failing checks
    if (complianceData.remediations.high) results.push(...complianceData.remediations.high);
    if (complianceData.remediations.medium) results.push(...complianceData.remediations.medium);
    if (complianceData.remediations.low) results.push(...complianceData.remediations.low);

    // Passing checks
    if (complianceData.passing_checks.high) results.push(...complianceData.passing_checks.high);
    if (complianceData.passing_checks.medium) results.push(...complianceData.passing_checks.medium);
    if (complianceData.passing_checks.low) results.push(...complianceData.passing_checks.low);

    // Manual checks
    if (complianceData.manual_checks) results.push(...complianceData.manual_checks);

    return results;
  }, [complianceData]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Results</h1>
          <p className="text-sm text-gray-500 mt-1">
            Detailed compliance check results
            {complianceData?.scan_date && (
              <span> &middot; Last scan: {new Date(complianceData.scan_date).toLocaleString()}</span>
            )}
          </p>
        </div>
        <button className="btn btn-secondary" onClick={refresh}>
          Refresh
        </button>
      </div>

      {allResults.length > 0 ? (
        <ResultsTable
          results={allResults}
          initialSeverity={initialSeverity}
          remediationNames={remediationNames}
          onRowClick={(name) => setDrawerCheckName(name)}
        />
      ) : (
        <div className="card p-12 text-center">
          <FileSearch className="h-12 w-12 text-gray-300 mx-auto mb-3" />
          <p className="text-gray-500">No compliance results available.</p>
          <p className="text-sm text-gray-400 mt-1">Run a scan from the Dashboard page to generate results.</p>
        </div>
      )}
      <CheckDetailDrawer
        checkName={drawerCheckName}
        onClose={() => setDrawerCheckName(null)}
      />
    </div>
  );
}
