import { Outlet, Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, FileSearch, Radar, Shield, Settings } from 'lucide-react';
import StatusIndicator from './StatusIndicator';
import { useDashboardStore } from '../lib/store';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Scans', href: '/scans', icon: Radar },
  { name: 'Results', href: '/results', icon: FileSearch },
  { name: 'Remediation', href: '/remediation', icon: Shield },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export default function Layout() {
  const location = useLocation();
  const { wsConnected, clusterStatus } = useDashboardStore();

  const isActive = (path: string) =>
    location.pathname === path || location.pathname.startsWith(path + '/');

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Sidebar */}
      <div className="fixed inset-y-0 left-0 w-60 bg-white border-r border-gray-200 z-30">
        <div className="flex flex-col h-full">
          {/* Logo */}
          <div className="px-5 py-4 border-b border-gray-200">
            <div className="flex items-center gap-2.5">
              <Shield className="h-7 w-7 text-primary-600" />
              <div>
                <span className="font-bold text-gray-900 text-sm leading-tight block">Compliance</span>
                <span className="text-xs text-gray-500 leading-tight block">Operator Dashboard</span>
              </div>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 px-3 py-4 space-y-0.5">
            {navigation.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    isActive(item.href)
                      ? 'bg-primary-50 text-primary-700'
                      : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                  }`}
                >
                  <Icon className="h-[18px] w-[18px]" />
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* Status footer */}
          <div className="p-4 border-t border-gray-200 space-y-2">
            <StatusIndicator
              connected={clusterStatus?.connected ?? false}
              label={clusterStatus?.connected ? 'Cluster connected' : 'Cluster disconnected'}
            />
            <StatusIndicator
              connected={wsConnected}
              label={wsConnected ? 'Live updates active' : 'Live updates offline'}
            />
          </div>
        </div>
      </div>

      {/* Main content */}
      <div className="pl-60">
        <main className="p-6 max-w-7xl">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
