import { describe, it, expect, beforeEach } from 'vitest';
import { useDashboardStore } from './store';
import type {
  ClusterStatus,
  OperatorStatus,
  InstallProgress,
  WSMessage,
} from '../types/api';

// Reset store between tests
beforeEach(() => {
  useDashboardStore.setState({
    wsConnected: false,
    clusterStatus: null,
    operatorStatus: null,
    complianceData: null,
    installProgress: [],
    uninstallProgress: [],
    updateCounter: 0,
  });
});

describe('setWSConnected', () => {
  it('sets wsConnected to true', () => {
    useDashboardStore.getState().setWSConnected(true);
    expect(useDashboardStore.getState().wsConnected).toBe(true);
  });

  it('sets wsConnected to false', () => {
    useDashboardStore.getState().setWSConnected(true);
    useDashboardStore.getState().setWSConnected(false);
    expect(useDashboardStore.getState().wsConnected).toBe(false);
  });
});

describe('setClusterStatus', () => {
  it('updates clusterStatus', () => {
    const status: ClusterStatus = {
      connected: true,
      server_url: 'https://api.test',
      arm_nodes: 0,
    };
    useDashboardStore.getState().setClusterStatus(status);
    expect(useDashboardStore.getState().clusterStatus).toEqual(status);
  });
});

describe('setOperatorStatus', () => {
  it('updates operatorStatus', () => {
    const status: OperatorStatus = { installed: true, version: '1.0.0' };
    useDashboardStore.getState().setOperatorStatus(status);
    expect(useDashboardStore.getState().operatorStatus).toEqual(status);
  });
});

describe('incrementUpdateCounter', () => {
  it('increments by 1 each call', () => {
    expect(useDashboardStore.getState().updateCounter).toBe(0);
    useDashboardStore.getState().incrementUpdateCounter();
    expect(useDashboardStore.getState().updateCounter).toBe(1);
    useDashboardStore.getState().incrementUpdateCounter();
    expect(useDashboardStore.getState().updateCounter).toBe(2);
  });
});

describe('installProgress', () => {
  it('addInstallProgress appends to array', () => {
    const p1: InstallProgress = { step: 'step1', message: 'Starting', done: false };
    const p2: InstallProgress = { step: 'step2', message: 'Done', done: true };

    useDashboardStore.getState().addInstallProgress(p1);
    expect(useDashboardStore.getState().installProgress).toHaveLength(1);

    useDashboardStore.getState().addInstallProgress(p2);
    expect(useDashboardStore.getState().installProgress).toHaveLength(2);
    expect(useDashboardStore.getState().installProgress[1]).toEqual(p2);
  });

  it('clearInstallProgress resets to empty array', () => {
    useDashboardStore.getState().addInstallProgress({ step: 's', message: 'm', done: false });
    useDashboardStore.getState().clearInstallProgress();
    expect(useDashboardStore.getState().installProgress).toEqual([]);
  });
});

describe('handleWSMessage', () => {
  it('cluster_status sets clusterStatus', () => {
    const payload: ClusterStatus = { connected: true, arm_nodes: 2 };
    const msg: WSMessage = { type: 'cluster_status', payload };

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().clusterStatus).toEqual(payload);
  });

  it('operator_status sets operatorStatus', () => {
    const payload: OperatorStatus = { installed: true, version: '2.0' };
    const msg: WSMessage = { type: 'operator_status', payload };

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().operatorStatus).toEqual(payload);
  });

  it('install_progress appends and does not increment counter when not done', () => {
    const payload: InstallProgress = { step: 'installing', message: 'In progress', done: false };
    const msg: WSMessage = { type: 'install_progress', payload };

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().installProgress).toHaveLength(1);
    expect(useDashboardStore.getState().updateCounter).toBe(0);
  });

  it('install_progress increments counter when done', () => {
    const payload: InstallProgress = { step: 'complete', message: 'Done', done: true };
    const msg: WSMessage = { type: 'install_progress', payload };

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().installProgress).toHaveLength(1);
    expect(useDashboardStore.getState().updateCounter).toBe(1);
  });

  it('uninstall_progress clears operator status on success (done, no error)', () => {
    // Set initial operator status
    useDashboardStore.getState().setOperatorStatus({ installed: true, version: '1.0' });

    const payload: InstallProgress = { step: 'done', message: 'Uninstalled', done: true };
    const msg: WSMessage = { type: 'uninstall_progress', payload };

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().operatorStatus).toEqual({ installed: false });
    expect(useDashboardStore.getState().updateCounter).toBe(1);
  });

  it('uninstall_progress does not clear operator status on error', () => {
    const original: OperatorStatus = { installed: true, version: '1.0' };
    useDashboardStore.getState().setOperatorStatus(original);

    const payload: InstallProgress = { step: 'done', message: 'Failed', done: true, error: 'timeout' };
    const msg: WSMessage = { type: 'uninstall_progress', payload };

    useDashboardStore.getState().handleWSMessage(msg);
    // operatorStatus should not have been reset to { installed: false }
    expect(useDashboardStore.getState().operatorStatus).toEqual(original);
    expect(useDashboardStore.getState().updateCounter).toBe(1);
  });

  it('check_result increments updateCounter', () => {
    const msg: WSMessage = { type: 'check_result', payload: {} };
    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().updateCounter).toBe(1);
  });

  it('scan_status increments updateCounter', () => {
    const msg: WSMessage = { type: 'scan_status', payload: {} };

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().updateCounter).toBe(1);
  });

  it('error does not change state (except console.error)', () => {
    const msg: WSMessage = { type: 'error', payload: 'something broke' };
    const counterBefore = useDashboardStore.getState().updateCounter;

    useDashboardStore.getState().handleWSMessage(msg);
    expect(useDashboardStore.getState().updateCounter).toBe(counterBefore);
    expect(useDashboardStore.getState().clusterStatus).toBeNull();
    expect(useDashboardStore.getState().operatorStatus).toBeNull();
  });
});
