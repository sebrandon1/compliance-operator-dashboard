import axios from 'axios';
import type {
  APIResponse,
  ClusterStatus,
  ComplianceData,
  OperatorStatus,
  CheckResult,
  CheckResultDetail,
  Summary,
  SuiteStatus,
  ProfileInfo,
  RemediationInfo,
  RemediationDetail,
  RemediationResult,
  ScanOptions,
} from '../types/api';

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

function unwrap<T>(response: { data: APIResponse<T> }): T {
  if (!response.data.success) {
    throw new Error(response.data.error || 'Unknown error');
  }
  return response.data.data as T;
}

export const clusterApi = {
  getStatus: async (): Promise<ClusterStatus> =>
    unwrap(await api.get('/cluster/status')),
};

export const operatorApi = {
  getStatus: async (): Promise<OperatorStatus> =>
    unwrap(await api.get('/operator/status')),

  install: async (): Promise<{ message: string }> =>
    unwrap(await api.post('/operator/install')),
};

export const scanApi = {
  create: async (opts: ScanOptions): Promise<{ message: string; name: string }> =>
    unwrap(await api.post('/scans', opts)),

  list: async (): Promise<SuiteStatus[]> =>
    unwrap(await api.get('/scans')),

  listProfiles: async (): Promise<ProfileInfo[]> =>
    unwrap(await api.get('/profiles')),
};

export const resultsApi = {
  getAll: async (): Promise<ComplianceData> =>
    unwrap(await api.get('/results')),

  getSummary: async (): Promise<Summary> =>
    unwrap(await api.get('/results/summary')),

  getFiltered: async (params: {
    severity?: string;
    status?: string;
    search?: string;
  }): Promise<CheckResult[]> =>
    unwrap(await api.get('/results', { params })),

  getDetail: async (name: string): Promise<CheckResultDetail> =>
    unwrap(await api.get(`/results/${encodeURIComponent(name)}`)),
};

export const remediationApi = {
  list: async (): Promise<RemediationInfo[]> =>
    unwrap(await api.get('/remediations')),

  getDetail: async (name: string): Promise<RemediationDetail> =>
    unwrap(await api.get(`/remediations/${encodeURIComponent(name)}`)),

  apply: async (name: string): Promise<RemediationResult> =>
    unwrap(await api.post(`/remediate/${encodeURIComponent(name)}`)),
};

export default api;
