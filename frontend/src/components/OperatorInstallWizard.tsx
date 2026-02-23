import { useState } from 'react';
import { CheckCircle, XCircle, Loader2, Download } from 'lucide-react';
import { operatorApi } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import type { InstallProgress } from '../types/api';

export default function OperatorInstallWizard() {
  const [installStarted, setInstallStarted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { installProgress, clearInstallProgress } = useDashboardStore();

  const handleInstall = async () => {
    setInstallStarted(true);
    setError(null);
    clearInstallProgress();

    try {
      await operatorApi.install();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start installation');
      setInstallStarted(false);
    }
  };

  // Derive installation status from progress
  const isComplete = installProgress.some(p => p.done && !p.error);
  const hasFailed = installProgress.some(p => p.done && !!p.error);
  const installing = installStarted && !isComplete && !hasFailed;

  const getStepIcon = (step: InstallProgress) => {
    if (step.error) return <XCircle className="h-5 w-5 text-red-500" />;
    if (step.done) return <CheckCircle className="h-5 w-5 text-emerald-500" />;
    return <Loader2 className="h-5 w-5 text-primary-500 animate-spin" />;
  };

  return (
    <div className="card">
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="font-semibold text-gray-900">Install Compliance Operator</h3>
        <p className="text-sm text-gray-500 mt-1">
          Install the OpenShift Compliance Operator into your cluster
        </p>
      </div>

      <div className="p-6">
        {installProgress.length === 0 && !installing && (
          <div className="text-center">
            <Download className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <p className="text-sm text-gray-600 mb-4">
              This will install the Compliance Operator, create the required namespace,
              set up RBAC, and wait for ProfileBundles to become valid.
            </p>
            <button
              className="btn btn-primary"
              onClick={handleInstall}
              disabled={installing}
            >
              Start Installation
            </button>
          </div>
        )}

        {(installProgress.length > 0 || installing) && (
          <div className="space-y-3">
            {installProgress.map((step, idx) => (
              <div key={idx} className="flex items-start gap-3">
                {getStepIcon(step)}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 capitalize">
                    {step.step.replace(/_/g, ' ')}
                  </p>
                  <p className={`text-xs mt-0.5 ${step.error ? 'text-red-600' : 'text-gray-500'}`}>
                    {step.message}
                  </p>
                </div>
              </div>
            ))}

            {installing && !isComplete && !hasFailed && (
              <div className="flex items-center gap-3 text-gray-500">
                <Loader2 className="h-5 w-5 animate-spin" />
                <span className="text-sm">Installing...</span>
              </div>
            )}
          </div>
        )}

        {error && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        {isComplete && (
          <div className="mt-4 p-3 bg-emerald-50 border border-emerald-200 rounded-lg">
            <p className="text-sm text-emerald-700 font-medium">
              Compliance Operator installed successfully!
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
