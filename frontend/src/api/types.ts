export type NodeStatus = "online" | "offline";
export type TaskStatus = "pending" | "running" | "success" | "failed";
export type EgressMode = "auto" | "ipv4" | "ipv6" | "custom";
export type PoolStrategy = "round" | "random" | "rand";

export interface Settings {
  proxyUsername: string;
  proxyPassword?: string;
}

export interface Node {
  id: string;
  name: string;
  publicIp: string;
  hostname: string;
  os: string;
  arch: string;
  status: NodeStatus;
  lastSeenAt: string;
  agentToken?: string;
  agentVersion: string;
  gostVersion: string;
  gostStatus: string;
  configVersion: number;
  groupIds: string[];
  httpPort: number;
  socksPort: number;
  egressMode: EgressMode | "";
  egressInterface: string;
  trafficDate?: string;
  todayUploadBytes: number;
  todayDownloadBytes: number;
  totalUploadBytes: number;
  totalDownloadBytes: number;
  createdAt: string;
  updatedAt: string;
}

export interface Group {
  id: string;
  name: string;
  remark: string;
  createdAt: string;
  updatedAt: string;
}

export interface Pool {
  id: string;
  name: string;
  groupIds: string[];
  httpPort: number;
  socksPort: number;
  strategy: PoolStrategy;
  enabled: boolean;
  runtimeStatus: string;
  runtimeError: string;
  startedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface RegisterToken {
  token: string;
  name: string;
  expiresAt: string;
  used: boolean;
  createdAt: string;
  installCommand: string;
}

export interface Task {
  id: string;
  nodeId: string;
  type: string;
  status: TaskStatus;
  payload: string;
  result: string;
  error: string;
  createdAt: string;
  updatedAt: string;
  startedAt: string;
  finishedAt: string;
}

export interface Versions {
  panel: string;
  agent: string;
}

export interface Summary {
  totalNodes: number;
  onlineNodes: number;
  gostActiveNodes: number;
  runningPools: number;
  failedTasks: number;
  outdatedAgentNodes: number;
  recentErrorTasks: Task[];
}

export interface State {
  nodes: Node[];
  groups: Group[];
  pools: Pool[];
  registerTokens: RegisterToken[];
  tasks: Task[];
  settings: Settings;
  baseURL: string;
  versions: Versions;
  summary: Summary;
}
