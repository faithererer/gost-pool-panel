# Frontend API Guide

This document is the handoff contract for building a modern frontend for GOST Pool Panel.

The backend is a Go single process. The old server-rendered pages still exist for compatibility, but new UI work should use the JSON APIs below.

## Product Context

GOST Pool Panel manages many Linux VPS nodes as a GOST proxy pool:

- The panel creates one-line Linux install commands.
- Nodes run `gost-pool-agent`, register, heartbeat, poll tasks, and apply GOST configs.
- The panel creates proxy pools on the management server, using node-side GOST HTTP proxies as upstreams.
- Users need to operate this without SSH in normal workflows.

Primary UI language is Chinese. The UI should feel like a practical operations console: dense, clear, fast, and safe around destructive actions.

## Authentication

Session auth uses an HTTP-only cookie named `gpp_session`.

Frontend requests must include cookies:

```ts
fetch("/api/admin/state", { credentials: "include" })
```

Error format:

```json
{ "error": "human readable message" }
```

Unauthenticated admin API requests return HTTP `401`.

### Login

`POST /api/admin/login`

Request:

```json
{ "username": "admin", "password": "admin123" }
```

Response:

```json
{ "authenticated": true, "user": "admin" }
```

### Logout

`POST /api/admin/logout`

Response:

```json
{ "ok": true }
```

### Session

`GET /api/admin/session`

Response:

```json
{ "authenticated": true, "user": "admin" }
```

## Core Types

```ts
export type NodeStatus = "online" | "offline";
export type TaskStatus = "pending" | "running" | "success" | "failed";
export type EgressMode = "auto" | "ipv4" | "ipv6" | "custom";
export type PoolStrategy = "round" | "random" | "rand";

export interface Settings {
  proxyUsername: string;
  proxyPassword: string;
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
  agentToken?: "";
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
```

Notes:

- `nodes[].agentToken` is intentionally blank in admin responses.
- `settings.proxyPassword` is returned for the single-admin MVP. Mask it by default in the UI.
- Timestamps are JSON `time.Time` strings. Zero values may appear as `0001-01-01T00:00:00Z`.

## State and Version

### Initial State

`GET /api/admin/state`

Returns `State`. This is the preferred first request after login.

### Version

`GET /api/admin/version`

Response:

```json
{ "panel": "0.3.5", "agent": "0.3.5" }
```

Use this to mark nodes whose `agentVersion !== versions.agent` as upgrade candidates.

## Tokens and Install Commands

### Create Register Token

`POST /api/admin/register-tokens`

Request:

```json
{ "name": "hk-01", "ttlHours": 24 }
```

Response: `RegisterToken`, including `installCommand`.

### Generate Install Command

`GET /api/admin/install-command?token=rt_xxx&name=hk-01`

Response:

```json
{ "command": "curl -fsSL http://panel/install.sh | bash -s -- --server http://panel --token rt_xxx --name 'hk-01'" }
```

Prefer using `registerTokens[].installCommand` from `/state` after token creation.

### Delete Register Token

`DELETE /api/admin/register-tokens/{token}`

Deletes the token record from the panel. It does not affect nodes that have already registered.

Response:

```json
{ "ok": true }
```

## Nodes

### List Nodes

`GET /api/admin/nodes`

Response: `Node[]`.

### Assign Node Groups

`POST /api/admin/nodes/{nodeId}/groups`

Request:

```json
{ "groupIds": ["group_a", "group_b"] }
```

Response: updated `Node`.

### Delete Node Record

`DELETE /api/admin/nodes/{nodeId}`

Only deletes panel data and related tasks. It does not operate the VPS.

Response:

```json
{ "ok": true }
```

### Cleanup Uninstalled Nodes

`POST /api/admin/nodes/cleanup-uninstalled`

Deletes all nodes with `gostStatus === "agent uninstalled"` and related tasks.

Response:

```json
{ "ok": true, "count": 3 }
```

## Node Tasks

Supported task types:

- `sync_node_proxy`
- `restart_gost`
- `apply_config`
- `update_ports`
- `upgrade_agent`
- `uninstall_agent`

### Create Single Node Task

`POST /api/admin/nodes/{nodeId}/tasks`

Common request:

```json
{
  "type": "upgrade_agent"
}
```

Sync proxy request:

```json
{
  "type": "sync_node_proxy",
  "httpPort": 18080,
  "socksPort": 18081,
  "gostVersion": "3.2.6",
  "egressMode": "ipv6",
  "egressInterface": ""
}
```

Rules:

- `sync_node_proxy` and `update_ports` generate the GOST payload server-side using global proxy credentials.
- `update_ports` is normalized to a `sync_node_proxy` task.
- `egressMode` supports `auto`, `ipv4`, `ipv6`, `custom`.
- `custom` requires `egressInterface`, such as `eth0` or `2600:...`.
- `apply_config` requires raw GOST JSON in `payload`.

Response: created `Task`.

### Create Batch Node Tasks

`POST /api/admin/nodes/tasks`

Request:

```json
{
  "nodeIds": ["node_a", "node_b"],
  "type": "upgrade_agent"
}
```

Response: `Task[]`.

Use this for batch upgrade, batch restart, batch uninstall, and batch sync.

### Retry Task

`POST /api/admin/tasks/{taskId}/retry`

Creates a new pending task with the same node, type, and payload.

Response: created `Task`.

### List Tasks

`GET /api/admin/tasks`

Response: `Task[]`.

### Delete Task Record

`DELETE /api/admin/tasks/{taskId}`

Deletes a task record. If you delete a `pending` task before the agent polls it, the agent will not execute it.

Response:

```json
{ "ok": true }
```

### Cleanup Tasks By Status

`POST /api/admin/tasks/cleanup`

Request:

```json
{ "statuses": ["success", "failed"] }
```

Response:

```json
{ "ok": true, "count": 12 }
```

## Groups

### List Groups

`GET /api/admin/groups`

Response: `Group[]`.

### Create Group

`POST /api/admin/groups`

Request:

```json
{ "name": "US", "remark": "Residential IPv6" }
```

Response: created `Group`.

### Update Group

`PATCH /api/admin/groups/{groupId}`

Request:

```json
{ "name": "US Home", "remark": "AT&T IPv6 nodes" }
```

Response: updated `Group`.

### Delete Group

`DELETE /api/admin/groups/{groupId}`

The backend removes this group ID from nodes and pools, then restarts enabled pools.

Response:

```json
{ "ok": true }
```

## Pools

### List Pools

`GET /api/admin/pools`

Response: `Pool[]`.

### Create Pool

`POST /api/admin/pools`

Request:

```json
{
  "name": "ai-pool",
  "groupIds": ["group_us"],
  "httpPort": 28080,
  "socksPort": 28081,
  "strategy": "round"
}
```

Response: created `Pool`. The backend attempts to start the pool runtime; if no active nodes exist, `runtimeStatus` and `runtimeError` explain why.

### Update Pool

`PATCH /api/admin/pools/{poolId}`

Partial request. Send only fields that changed:

```json
{
  "name": "ai-pool-v2",
  "groupIds": ["group_us", "group_hk"],
  "httpPort": 28080,
  "socksPort": 28081,
  "strategy": "random",
  "enabled": true
}
```

If enabled, the backend restarts the runtime. If disabled, the backend stops it and sets `runtimeStatus` to `disabled`.

Response: updated `Pool`.

### Restart Pool Runtime

`POST /api/admin/pools/{poolId}/restart`

Response: updated `Pool`, or HTTP `400` with `{ "error": "..." }` if it cannot start.

### Delete Pool

`DELETE /api/admin/pools/{poolId}`

The backend stops the runtime first, then deletes the record.

Response:

```json
{ "ok": true }
```

## Settings

### Update Proxy Auth

`PATCH /api/admin/settings`

Request:

```json
{ "proxyUsername": "proxy", "proxyPassword": "secret" }
```

Behavior:

- Updates global proxy credentials.
- Automatically creates `sync_node_proxy` tasks for non-uninstalled nodes.
- Restarts enabled proxy pools.

Response: updated `Settings`.

## UI State Mapping

Node status:

- `online`: green, primary healthy state.
- `offline`: gray or muted.

GOST status:

- contains `active`: healthy.
- `not installed`: setup incomplete.
- contains `inactive`, `failed`, or `exited`: warning/error.
- `agent uninstalled`: historical record; offer cleanup/delete.

Task status:

- `pending`: queued.
- `running`: in progress.
- `success`: completed.
- `failed`: error; show `error` prominently and allow retry.

Pool runtime:

- `running`: healthy.
- `no active nodes`: pool exists but no usable upstream node.
- `gost not found`: management host/container lacks `gost`.
- `config failed`, `start failed`, `exited`: show `runtimeError`.
- `disabled`: user disabled the pool.

## Recommended Pages

### Dashboard

Use `/api/admin/state`.

Show:

- Online nodes / total nodes.
- GOST active nodes.
- Running pools.
- Failed tasks.
- Outdated agent nodes.
- Recent failed tasks.
- Quick actions: create token, sync all, upgrade outdated agents.

### Nodes

Features:

- Search by name, IP, hostname.
- Filter by status, group, agent version, GOST status, egress mode.
- Batch actions: upgrade agent, sync proxy, restart GOST, assign group.
- Row actions: details, sync proxy, upgrade agent, restart GOST, uninstall agent, delete record.

Node detail should show:

- System and version.
- HTTP/SOCKS ports.
- Egress mode and interface.
- Groups.
- Recent tasks.
- Traffic totals.

### Tokens

Features:

- Create token.
- Copy install command.
- Display used/available/expired.
- Explain that tokens are one-time for new nodes; existing nodes can in-place upgrade with saved identity.

### Groups

Features:

- Create/edit/delete groups.
- Show node count and pool usage by deriving from state.
- Confirm delete because it removes references from nodes and pools.

### Pools

Features:

- Create/edit/delete pools.
- Restart runtime.
- Copy test commands:

```bash
curl -x http://HOST:HTTP_PORT -U 'USER:PASS' https://api64.ipify.org
curl -x socks5h://HOST:SOCKS_PORT -U 'USER:PASS' https://api64.ipify.org
```

- For IPv6 panel hosts, use bracketed addresses: `[IPv6]:PORT`.
- Show available upstream nodes from selected groups.

### Tasks

Features:

- Filter by status, type, node.
- Expand payload/result/error.
- Retry failed tasks.
- Highlight tasks older than expected still pending/running.

### Settings

Features:

- Edit global proxy auth.
- Mask proxy password by default.
- Show panel version and bundled agent version.
- Show base URL.
- Include dangerous operations in a visually separate area.

## Critical Interactions

- Copy install command: use `navigator.clipboard.writeText`.
- Upgrade outdated agents: call batch task API with nodes whose `agentVersion !== versions.agent`.
- Sync IPv6 node proxy: send `egressMode: "ipv6"` and leave `egressInterface` empty unless the user chooses custom.
- Custom egress: require interface/IP input before submitting.
- Delete node/group/pool and uninstall agent: always require confirmation.
- Failed API response: show the `{error}` body exactly enough for troubleshooting.

## Suggested Frontend Build Order

1. API client, login, session guard.
2. Global layout and `/api/admin/state` loading.
3. Dashboard.
4. Nodes with single-node task actions.
5. Tokens and install command copy.
6. Groups.
7. Pools and test command copy.
8. Tasks with retry.
9. Settings.
10. Batch actions, filters, empty states, mobile layout polish.

## Acceptance Checklist

- Login works and unauthenticated state fetch receives `401`.
- Refreshing the browser keeps the session.
- User can create a token and copy a working install command.
- User can assign a node to groups.
- User can sync node proxy with `auto`, `ipv4`, `ipv6`, and `custom`.
- User can batch upgrade agents.
- User can create, update, restart, disable, and delete pools.
- Pool test commands are copyable and use `api64.ipify.org`.
- Failed tasks show error details and can be retried.
- Settings update triggers node sync and pool restart.
- Destructive actions require confirmation.
