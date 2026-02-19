// Matches Go types from internal/compliance/types.go

export type Severity = 'high' | 'medium' | 'low';
export type CheckStatus = 'PASS' | 'FAIL' | 'MANUAL' | 'SKIP' | 'NOT-APPLICABLE';

export interface CheckResult {
  name: string;
  check: string;
  status: CheckStatus;
  description: string;
  severity: Severity;
  scan_name?: string;
  suite?: string;
}

export interface CheckResultDetail {
  name: string;
  check: string;
  status: CheckStatus;
  description: string;
  severity: Severity;
  scan_name?: string;
  suite?: string;
  id: string;
  instructions: string;
  rationale: string;
  has_remediation: boolean;
  remediation_name?: string;
}

export interface Summary {
  total_checks: number;
  passing: number;
  failing: number;
  manual: number;
  skipped: number;
}

export interface SeverityMap {
  high: CheckResult[];
  medium: CheckResult[];
  low: CheckResult[];
}

export interface ComplianceData {
  scan_date: string;
  summary: Summary;
  remediations: SeverityMap;
  passing_checks: SeverityMap;
  manual_checks: CheckResult[];
}

export interface PodStatus {
  name: string;
  phase: string;
  ready: boolean;
  reason?: string;
}

export interface BundleStatus {
  name: string;
  data_stream_status: string;
}

export interface OperatorStatus {
  installed: boolean;
  version?: string;
  csv_phase?: string;
  pods?: PodStatus[];
  profile_bundles?: BundleStatus[];
}

export interface InstallProgress {
  step: string;
  message: string;
  done: boolean;
  error?: string;
}

export interface ProfileInfo {
  name: string;
  title: string;
  description?: string;
}

export interface ScanOptions {
  name: string;
  profile: string;
  namespace?: string;
}

export interface ScanStatus {
  name: string;
  phase: string;
  result?: string;
  profile?: string;
  scan_type?: string;
  content_image?: string;
  start_timestamp?: string;
  end_timestamp?: string;
  warnings?: string;
}

export interface Condition {
  type: string;
  status: string;
  reason?: string;
  last_transition_time?: string;
}

export interface SuiteStatus {
  name: string;
  phase: string;
  scans?: ScanStatus[];
  result?: string;
  created_at?: string;
  conditions?: Condition[];
}

export interface RemediationInfo {
  name: string;
  kind: string;
  severity: Severity;
  applied: boolean;
  reboot_needed: boolean;
  role?: string;
}

export interface RemediationDetail {
  name: string;
  kind: string;
  severity: Severity;
  applied: boolean;
  reboot_needed: boolean;
  role?: string;
  object_yaml: string;
  api_version?: string;
  namespace?: string;
}

export interface RemediationResult {
  name: string;
  applied: boolean;
  message: string;
  error?: string;
}

export interface StorageInfo {
  has_default_storage_class: boolean;
  storage_class_name?: string;
  provisioner?: string;
  hostpath_csi_deployed: boolean;
  recommendation?: string;
}

export interface ClusterStatus {
  connected: boolean;
  server_url?: string;
  server_version?: string;
  platform?: string;
  architecture?: string;
  arm_nodes: number;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

// WebSocket message types
export type WSMessageType =
  | 'cluster_status'
  | 'operator_status'
  | 'install_progress'
  | 'scan_status'
  | 'check_result'
  | 'remediation'
  | 'remediation_result'
  | 'error';

export interface WSMessage {
  type: WSMessageType;
  payload: unknown;
}

export interface WatchEvent {
  event_type: 'ADDED' | 'MODIFIED' | 'DELETED';
  resource_type: string;
  name: string;
  namespace: string;
  data?: Record<string, unknown>;
}
