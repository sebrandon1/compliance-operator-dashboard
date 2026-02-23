import { vi, describe, it, expect, beforeEach } from 'vitest';

// Declare mocks with vi.hoisted so they're available when vi.mock factory runs
const { mockGet, mockPost, mockDelete } = vi.hoisted(() => ({
  mockGet: vi.fn(),
  mockPost: vi.fn(),
  mockDelete: vi.fn(),
}));

vi.mock('axios', () => ({
  default: {
    create: () => ({
      get: mockGet,
      post: mockPost,
      delete: mockDelete,
    }),
  },
}));

// Import after mocking
import {
  clusterApi,
  operatorApi,
  scanApi,
  resultsApi,
  remediationApi,
} from './api';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('unwrap helper (via API calls)', () => {
  it('returns data when success is true', async () => {
    const payload = { connected: true, server_url: 'https://api.test' };
    mockGet.mockResolvedValue({ data: { success: true, data: payload } });

    const result = await clusterApi.getStatus();
    expect(result).toEqual(payload);
  });

  it('throws when success is false with error message', async () => {
    mockGet.mockResolvedValue({
      data: { success: false, error: 'cluster unreachable' },
    });

    await expect(clusterApi.getStatus()).rejects.toThrow('cluster unreachable');
  });

  it('throws "Unknown error" when success is false with no error message', async () => {
    mockGet.mockResolvedValue({ data: { success: false } });

    await expect(clusterApi.getStatus()).rejects.toThrow('Unknown error');
  });
});

describe('clusterApi', () => {
  it('getStatus calls GET /cluster/status', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: {} } });
    await clusterApi.getStatus();
    expect(mockGet).toHaveBeenCalledWith('/cluster/status');
  });
});

describe('operatorApi', () => {
  it('getStatus calls GET /operator/status', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: {} } });
    await operatorApi.getStatus();
    expect(mockGet).toHaveBeenCalledWith('/operator/status');
  });

  it('install calls POST /operator/install', async () => {
    mockPost.mockResolvedValue({ data: { success: true, data: {} } });
    await operatorApi.install();
    expect(mockPost).toHaveBeenCalledWith('/operator/install');
  });

  it('uninstall calls DELETE /operator', async () => {
    mockDelete.mockResolvedValue({ data: { success: true, data: {} } });
    await operatorApi.uninstall();
    expect(mockDelete).toHaveBeenCalledWith('/operator');
  });
});

describe('scanApi', () => {
  it('create calls POST /scans with options', async () => {
    const opts = { name: 'my-scan', profile: 'cis' };
    mockPost.mockResolvedValue({ data: { success: true, data: {} } });
    await scanApi.create(opts);
    expect(mockPost).toHaveBeenCalledWith('/scans', opts);
  });

  it('createRecommended calls POST /scans/recommended', async () => {
    mockPost.mockResolvedValue({ data: { success: true, data: {} } });
    await scanApi.createRecommended();
    expect(mockPost).toHaveBeenCalledWith('/scans/recommended');
  });

  it('list calls GET /scans', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: [] } });
    await scanApi.list();
    expect(mockGet).toHaveBeenCalledWith('/scans');
  });

  it('listProfiles calls GET /profiles', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: [] } });
    await scanApi.listProfiles();
    expect(mockGet).toHaveBeenCalledWith('/profiles');
  });

  it('rescan URL-encodes the scan name', async () => {
    mockPost.mockResolvedValue({ data: { success: true, data: {} } });
    await scanApi.rescan('my scan/test');
    expect(mockPost).toHaveBeenCalledWith(
      `/scans/${encodeURIComponent('my scan/test')}/rescan`
    );
  });

  it('delete URL-encodes the scan name', async () => {
    mockDelete.mockResolvedValue({ data: { success: true, data: {} } });
    await scanApi.delete('scan&special=chars');
    expect(mockDelete).toHaveBeenCalledWith(
      `/scans/${encodeURIComponent('scan&special=chars')}`
    );
  });
});

describe('resultsApi', () => {
  it('getAll calls GET /results', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: {} } });
    await resultsApi.getAll();
    expect(mockGet).toHaveBeenCalledWith('/results');
  });

  it('getSummary calls GET /results/summary', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: {} } });
    await resultsApi.getSummary();
    expect(mockGet).toHaveBeenCalledWith('/results/summary');
  });

  it('getFiltered calls GET /results with params', async () => {
    const params = { severity: 'high', status: 'FAIL' };
    mockGet.mockResolvedValue({ data: { success: true, data: [] } });
    await resultsApi.getFiltered(params);
    expect(mockGet).toHaveBeenCalledWith('/results', { params });
  });

  it('getDetail URL-encodes the check name', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: {} } });
    await resultsApi.getDetail('check/with spaces');
    expect(mockGet).toHaveBeenCalledWith(
      `/results/${encodeURIComponent('check/with spaces')}`
    );
  });
});

describe('remediationApi', () => {
  it('list calls GET /remediations', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: [] } });
    await remediationApi.list();
    expect(mockGet).toHaveBeenCalledWith('/remediations');
  });

  it('getDetail URL-encodes the name', async () => {
    mockGet.mockResolvedValue({ data: { success: true, data: {} } });
    await remediationApi.getDetail('rem/name');
    expect(mockGet).toHaveBeenCalledWith(
      `/remediations/${encodeURIComponent('rem/name')}`
    );
  });

  it('apply calls POST /remediate/:name', async () => {
    mockPost.mockResolvedValue({ data: { success: true, data: {} } });
    await remediationApi.apply('my-rem');
    expect(mockPost).toHaveBeenCalledWith(
      `/remediate/${encodeURIComponent('my-rem')}`
    );
  });

  it('applyBatch calls POST /remediate with names', async () => {
    const names = ['rem-1', 'rem-2'];
    mockPost.mockResolvedValue({ data: { success: true, data: [] } });
    await remediationApi.applyBatch(names);
    expect(mockPost).toHaveBeenCalledWith('/remediate', { names });
  });

  it('remove calls DELETE /remediate/:name', async () => {
    mockDelete.mockResolvedValue({ data: { success: true, data: {} } });
    await remediationApi.remove('my-rem');
    expect(mockDelete).toHaveBeenCalledWith(
      `/remediate/${encodeURIComponent('my-rem')}`
    );
  });
});
