interface StatusIndicatorProps {
  connected: boolean;
  label?: string;
}

export default function StatusIndicator({ connected, label }: StatusIndicatorProps) {
  return (
    <div className="flex items-center gap-2">
      <span
        className={`inline-block h-2.5 w-2.5 rounded-full ${
          connected
            ? 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.6)]'
            : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.6)]'
        }`}
      />
      {label && (
        <span className="text-xs text-gray-500">
          {label}
        </span>
      )}
    </div>
  );
}
