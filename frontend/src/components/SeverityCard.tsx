import { useNavigate } from 'react-router-dom';
import { AlertTriangle, AlertCircle, Info } from 'lucide-react';
import type { Severity } from '../types/api';

interface SeverityCardProps {
  severity: Severity;
  failCount: number;
  passCount: number;
  total: number;
}

const severityConfig = {
  high: {
    icon: AlertTriangle,
    label: 'High',
    bgColor: 'bg-red-50',
    borderColor: 'border-red-200',
    iconColor: 'text-red-600',
    countColor: 'text-red-700',
    barColor: 'bg-red-500',
  },
  medium: {
    icon: AlertCircle,
    label: 'Medium',
    bgColor: 'bg-amber-50',
    borderColor: 'border-amber-200',
    iconColor: 'text-amber-600',
    countColor: 'text-amber-700',
    barColor: 'bg-amber-500',
  },
  low: {
    icon: Info,
    label: 'Low',
    bgColor: 'bg-blue-50',
    borderColor: 'border-blue-200',
    iconColor: 'text-blue-600',
    countColor: 'text-blue-700',
    barColor: 'bg-blue-500',
  },
};

export default function SeverityCard({ severity, failCount, passCount, total }: SeverityCardProps) {
  const navigate = useNavigate();
  const config = severityConfig[severity];
  const Icon = config.icon;
  const passRate = total > 0 ? Math.round((passCount / total) * 100) : 0;

  return (
    <div
      className={`card ${config.bgColor} border ${config.borderColor} p-5 cursor-pointer hover:shadow-md transition-shadow`}
      onClick={() => navigate(`/results?severity=${severity}`)}
    >
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Icon className={`h-5 w-5 ${config.iconColor}`} />
            <h3 className="font-semibold text-gray-900">{config.label} Severity</h3>
          </div>
          <div className="mt-3">
            <span className={`text-3xl font-bold ${config.countColor}`}>{failCount}</span>
            <span className="text-sm text-gray-500 ml-1.5">failing</span>
          </div>
          <div className="text-sm text-gray-500 mt-1">
            {passCount} passing / {total} total
          </div>
        </div>
        <div className="text-right">
          <span className="text-2xl font-bold text-gray-700">{passRate}%</span>
          <p className="text-xs text-gray-500">pass rate</p>
        </div>
      </div>

      {/* Progress bar */}
      <div className="mt-4 h-2 rounded-full bg-gray-200 overflow-hidden">
        <div
          className={`h-full rounded-full ${passCount > 0 ? 'bg-emerald-500' : ''}`}
          style={{ width: `${passRate}%` }}
        />
      </div>
    </div>
  );
}
