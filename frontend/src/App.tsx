import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import DashboardPage from './pages/DashboardPage';
import ResultsPage from './pages/ResultsPage';
import ScansPage from './pages/ScansPage';
import RemediationPage from './pages/RemediationPage';
import CheckDetailPage from './pages/CheckDetailPage';
import RemediationDetailPage from './pages/RemediationDetailPage';
import SettingsPage from './pages/SettingsPage';
import { useWebSocket } from './hooks/useWebSocket';
import { useCluster } from './hooks/useCluster';

function App() {
  useWebSocket();
  useCluster();

  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Navigate to="/dashboard" />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="scans" element={<ScansPage />} />
        <Route path="results" element={<ResultsPage />} />
        <Route path="results/:name" element={<CheckDetailPage />} />
        <Route path="remediation" element={<RemediationPage />} />
        <Route path="remediation/:name" element={<RemediationDetailPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  );
}

export default App;
