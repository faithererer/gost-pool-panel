import { useState } from 'react';
import { motion } from 'framer-motion';
import { Copy, PowerOff, RefreshCw, RotateCw, Search, Server, SlidersHorizontal, Terminal, Trash2, Wrench } from 'lucide-react';
import { api } from '../api';
import { useAppContext } from '../api/AppContext';
import { Node } from '../api/types';
import { Button } from '../components/ui/button';
import { Card, CardContent } from '../components/ui/card';
import { ConfirmDialog } from '../components/ui/confirm-dialog';
import { Input } from '../components/ui/input';
import { Modal } from '../components/ui/modal';
import { copyText } from '../lib/clipboard';
import { authenticatedProxyURL, ProxyProtocol, proxyTestCommand } from '../lib/proxy';

type SyncForm = {
  node: Node;
  httpPort: number;
  socksPort: number;
  gostVersion: string;
  egressMode: string;
  egressInterface: string;
};

const statusText = (status: string) => (status === 'online' ? '在线' : '离线');
const gostHealthy = (status: string) => status.toLowerCase().includes('active');

function formatBytes(value: number) {
  if (!value) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function proxyLabel(protocol: ProxyProtocol) {
  return protocol === 'http' ? 'HTTP' : 'SOCKS5';
}

function egressLabel(mode?: string) {
  switch (mode) {
    case 'ipv4':
      return '强制 IPv4';
    case 'ipv6':
      return '强制 IPv6';
    case 'prefer_ipv6':
      return 'IPv6 优先';
    case 'custom':
      return '自定义';
    default:
      return '自动';
  }
}

export default function Nodes() {
  const { state, refreshState, notify } = useAppContext();
  const [search, setSearch] = useState('');
  const [confirmDeleteNode, setConfirmDeleteNode] = useState<Node | null>(null);
  const [confirmUninstall, setConfirmUninstall] = useState<Node | null>(null);
  const [editingGroupsNode, setEditingGroupsNode] = useState<Node | null>(null);
  const [selectedGroups, setSelectedGroups] = useState<string[]>([]);
  const [syncForm, setSyncForm] = useState<SyncForm | null>(null);
  const [batchUpgrading, setBatchUpgrading] = useState(false);

  if (!state) return null;

  const filteredNodes = state.nodes.filter((node) => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) return true;
    return [node.name, node.publicIp, node.hostname, node.id]
      .some((value) => (value || '').toLowerCase().includes(keyword));
  });

  const handleAction = async (action: () => Promise<unknown>, successTitle?: string, successMessage?: string) => {
    try {
      await action();
      await refreshState();
      if (successTitle) {
        notify({ type: 'success', title: successTitle, message: successMessage });
      }
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '操作失败', message: error.response?.data?.error || '请稍后重试。' });
    }
  };

  const handleBatchUpgrade = async () => {
    const outdated = state.nodes
      .filter((node) => node.gostStatus !== 'agent uninstalled' && node.agentVersion !== state.versions.agent)
      .map((node) => node.id);
    if (outdated.length === 0) {
      notify({ type: 'info', title: '没有需要升级的节点', message: `当前内置 Agent 版本为 ${state.versions.agent}。` });
      return;
    }
    setBatchUpgrading(true);
    try {
      await handleAction(
        () => api.post('/nodes/tasks', { type: 'upgrade_agent', nodeIds: outdated }),
        '升级任务已下发',
        `${outdated.length} 个节点会在下次 Agent 轮询时开始升级。`
      );
    } finally {
      setBatchUpgrading(false);
    }
  };

  const handleAssignGroups = async () => {
    if (!editingGroupsNode) return;
    await handleAction(
      () => api.post(`/nodes/${editingGroupsNode.id}/groups`, { groupIds: selectedGroups }),
      '节点分组已保存'
    );
    setEditingGroupsNode(null);
  };

  const openSync = (node: Node) => {
    setSyncForm({
      node,
      httpPort: node.httpPort || 18080,
      socksPort: node.socksPort || 18081,
      gostVersion: node.gostVersion && node.gostVersion !== 'not installed' ? node.gostVersion : '3.2.6',
      egressMode: node.egressMode || 'auto',
      egressInterface: node.egressInterface || '',
    });
  };

  const handleSyncNode = async () => {
    if (!syncForm) return;
    await handleAction(
      () => api.post(`/nodes/${syncForm.node.id}/tasks`, {
        type: 'sync_node_proxy',
        httpPort: syncForm.httpPort,
        socksPort: syncForm.socksPort,
        gostVersion: syncForm.gostVersion,
        egressMode: syncForm.egressMode,
        egressInterface: syncForm.egressInterface,
      }),
      '同步任务已下发',
      `${syncForm.node.name} 会在下次 Agent 轮询时应用代理配置。`
    );
    setSyncForm(null);
  };

  const groupNames = (node: Node) => state.groups
    .filter((group) => node.groupIds?.includes(group.id))
    .map((group) => group.name);

  const syncableNodeIds = state.nodes
    .filter((node) => node.gostStatus !== 'agent uninstalled')
    .map((node) => node.id);

  const copyProxyText = async (text: string, title: string) => {
    const ok = await copyText(text);
    notify(ok
      ? { type: 'success', title }
      : { type: 'error', title: '复制失败', message: '浏览器限制了剪贴板访问，请手动复制。' });
  };

  const copyDirectAddress = async (node: Node, protocol: ProxyProtocol, port: number) => {
    if (!node.publicIp) {
      notify({ type: 'error', title: '缺少节点公网 IP', message: '面板还没有收到该节点的公网地址。' });
      return;
    }
    await copyProxyText(authenticatedProxyURL(protocol, node.publicIp, port, state.settings), `${node.name} ${proxyLabel(protocol)} 地址已复制`);
  };

  const copyDirectTestCommand = async (node: Node, protocol: ProxyProtocol, port: number) => {
    if (!node.publicIp) {
      notify({ type: 'error', title: '缺少节点公网 IP', message: '面板还没有收到该节点的公网地址。' });
      return;
    }
    await copyProxyText(
      proxyTestCommand(protocol, node.publicIp, port, state.settings),
      `${node.name} ${proxyLabel(protocol)} 测试命令已复制`
    );
  };

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="space-y-6">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">节点管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">管理 Linux Agent、节点代理端口、出口网络和远程任务。</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={handleBatchUpgrade} disabled={batchUpgrading}>
            <RefreshCw className="mr-2 h-4 w-4" />
            批量升级 Agent
          </Button>
          <Button
            variant="outline"
            onClick={() => handleAction(
              () => api.post('/nodes/cleanup-uninstalled'),
              '清理完成',
              '已卸载节点及其关联任务记录已从面板移除。'
            )}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            清理已卸载
          </Button>
          <Button
            onClick={() => {
              if (syncableNodeIds.length === 0) {
                notify({ type: 'info', title: '没有可同步的节点' });
                return;
              }
              void handleAction(
                () => api.post('/nodes/tasks', { type: 'sync_node_proxy', nodeIds: syncableNodeIds }),
                '同步任务已下发',
                `${syncableNodeIds.length} 个节点会在下次 Agent 轮询时应用代理配置。`
              );
            }}
          >
            <Wrench className="mr-2 h-4 w-4" />
            全量同步代理
          </Button>
        </div>
      </div>

      <div className="flex w-full max-w-sm items-center gap-2">
        <Search className="h-5 w-5 text-muted-foreground" />
        <Input type="search" placeholder="搜索名称、IP、主机名或节点 ID" value={search} onChange={(event) => setSearch(event.target.value)} />
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {filteredNodes.map((node, index) => {
          const groups = groupNames(node);
          const status = node.gostStatus || 'unknown';
          const directEndpoints = [
            { label: 'HTTP', protocol: 'http' as const, port: node.httpPort },
            { label: 'SOCKS5', protocol: 'socks5h' as const, port: node.socksPort },
          ].filter((endpoint) => endpoint.port > 0);
          return (
            <motion.div
              key={node.id}
              initial={{ opacity: 0, scale: 0.96 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.2, delay: index * 0.03 }}
            >
              <Card className="h-full transition-colors hover:border-primary/50">
                <CardContent className="flex h-full flex-col p-5">
                  <div className="mb-4 flex items-start justify-between gap-3">
                    <div className="flex min-w-0 items-center gap-3">
                      <div className={`rounded-full p-2 ${node.status === 'online' ? 'bg-green-500/10 text-green-500' : 'bg-muted text-muted-foreground'}`}>
                        <Server className="h-5 w-5" />
                      </div>
                      <div className="min-w-0">
                        <h3 className="truncate text-lg font-semibold" title={node.name}>{node.name}</h3>
                        <p className="truncate font-mono text-xs text-muted-foreground">{node.publicIp || node.hostname || node.id}</p>
                      </div>
                    </div>
                    <span className={`shrink-0 rounded-full px-2 py-1 text-xs font-medium ${node.status === 'online' ? 'bg-green-500/10 text-green-500' : 'bg-muted text-muted-foreground'}`}>
                      {statusText(node.status)}
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground">
                    <span className="truncate">系统：{node.os || '-'}</span>
                    <span className="truncate">架构：{node.arch || '-'}</span>
                    <span className="truncate">HTTP：{node.httpPort || '-'}</span>
                    <span className="truncate">SOCKS5：{node.socksPort || '-'}</span>
                    <span className="truncate">Agent：{node.agentVersion || 'unknown'}</span>
                    <span className={`truncate ${gostHealthy(status) ? 'text-green-500' : 'text-yellow-500'}`}>
                      GOST：{node.gostVersion || 'unknown'} {status}
                    </span>
                  </div>

                  <div className="mt-4 flex flex-wrap gap-2">
                    <span className="rounded-full bg-secondary px-2 py-1 text-xs text-secondary-foreground">
                      出口：{egressLabel(node.egressMode)}{node.egressInterface ? ` / ${node.egressInterface}` : ''}
                    </span>
                    {groups.length > 0 ? groups.map((name) => (
                      <span key={name} className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">{name}</span>
                    )) : (
                      <span className="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">未分组</span>
                    )}
                  </div>

                  <div className="mt-4 rounded-md border bg-muted/30 p-3 text-xs text-muted-foreground">
                    今日：{formatBytes(node.todayUploadBytes)} 上行 / {formatBytes(node.todayDownloadBytes)} 下行
                    <br />
                    累计：{formatBytes(node.totalUploadBytes)} 上行 / {formatBytes(node.totalDownloadBytes)} 下行
                  </div>

                  <div className="mt-4 rounded-md border bg-background/40 p-3">
                    <div className="mb-2 text-xs font-medium text-muted-foreground">直连代理</div>
                    {node.publicIp && directEndpoints.length > 0 ? (
                      <div className="space-y-2">
                        {directEndpoints.map((endpoint) => {
                          const address = authenticatedProxyURL(endpoint.protocol, node.publicIp, endpoint.port, state.settings);
                          return (
                            <div key={endpoint.protocol} className="grid grid-cols-[52px_minmax(0,1fr)_32px_74px] items-center gap-2 text-xs">
                              <span className="text-muted-foreground">{endpoint.label}</span>
                              <code className="truncate rounded bg-muted px-2 py-1 font-mono text-foreground" title={address}>{address}</code>
                              <Button variant="ghost" size="icon" className="h-8 w-8" title="复制含认证代理地址" onClick={() => copyDirectAddress(node, endpoint.protocol, endpoint.port)}>
                                <Copy className="h-3.5 w-3.5" />
                              </Button>
                              <Button variant="outline" size="sm" className="h-8 px-2 text-xs" onClick={() => copyDirectTestCommand(node, endpoint.protocol, endpoint.port)}>
                                <Terminal className="mr-1 h-3.5 w-3.5" />
                                测试
                              </Button>
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <p className="text-xs text-muted-foreground">暂无可用直连地址，等待节点上报公网 IP 和端口。</p>
                    )}
                  </div>

                  <div className="mt-auto flex flex-wrap gap-2 border-t border-border pt-4">
                    <Button variant="outline" size="sm" onClick={() => {
                      setEditingGroupsNode(node);
                      setSelectedGroups(node.groupIds || []);
                    }}>
                      分组
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => openSync(node)}>
                      <SlidersHorizontal className="mr-1 h-4 w-4" />
                      同步代理
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleAction(
                        () => api.post('/nodes/tasks', { type: 'restart_gost', nodeIds: [node.id] }),
                        '重启任务已下发',
                        `${node.name} 会在下次 Agent 轮询时重启 GOST。`
                      )}
                    >
                      <RotateCw className="mr-1 h-4 w-4" />
                      重启
                    </Button>
                    <Button variant="destructive" size="sm" onClick={() => setConfirmUninstall(node)}>
                      <PowerOff className="mr-1 h-4 w-4" />
                      卸载
                    </Button>
                    <Button variant="ghost" size="sm" className="text-destructive hover:bg-destructive/10 hover:text-destructive" onClick={() => setConfirmDeleteNode(node)}>
                      删除记录
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          );
        })}
      </div>

      {filteredNodes.length === 0 && (
        <div className="py-20 text-center text-muted-foreground">
          <Server className="mx-auto mb-4 h-12 w-12 opacity-50" />
          <p>没有找到匹配的节点。</p>
        </div>
      )}

      <ConfirmDialog
        isOpen={!!confirmDeleteNode}
        onClose={() => setConfirmDeleteNode(null)}
        title="确认删除节点记录？"
        description={`这只会删除面板中的 ${confirmDeleteNode?.name} 记录和关联任务，不会操作 VPS。`}
        onConfirm={() => handleAction(
          () => api.delete(`/nodes/${confirmDeleteNode?.id}`),
          '节点记录已删除'
        )}
      />

      <ConfirmDialog
        isOpen={!!confirmUninstall}
        onClose={() => setConfirmUninstall(null)}
        title="确认远程卸载 Agent？"
        description={`将向 ${confirmUninstall?.name} 下发卸载任务。成功后节点会离线，并可在面板中清理记录。`}
        confirmText="下发卸载"
        onConfirm={() => handleAction(
          () => api.post('/nodes/tasks', { type: 'uninstall_agent', nodeIds: [confirmUninstall?.id] }),
          '卸载任务已下发',
          `${confirmUninstall?.name} 会在下次 Agent 轮询时执行卸载。`
        )}
      />

      <Modal
        isOpen={!!editingGroupsNode}
        onClose={() => setEditingGroupsNode(null)}
        title={`编辑节点分组 - ${editingGroupsNode?.name}`}
        footer={(
          <>
            <Button variant="outline" onClick={() => setEditingGroupsNode(null)}>取消</Button>
            <Button onClick={handleAssignGroups}>保存</Button>
          </>
        )}
      >
        <div className="space-y-3 py-2">
          {state.groups.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无分组，请先到分组页面创建。</p>
          ) : state.groups.map((group) => (
            <label key={group.id} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                className="h-4 w-4 rounded border-input text-primary focus:ring-primary"
                checked={selectedGroups.includes(group.id)}
                onChange={(event) => {
                  setSelectedGroups(event.target.checked
                    ? [...selectedGroups, group.id]
                    : selectedGroups.filter((id) => id !== group.id));
                }}
              />
              {group.name}
            </label>
          ))}
        </div>
      </Modal>

      <Modal
        isOpen={!!syncForm}
        onClose={() => setSyncForm(null)}
        title={`同步节点代理 - ${syncForm?.node.name}`}
        description="修改端口或出口网络后会下发 sync_node_proxy 任务。"
        footer={(
          <>
            <Button variant="outline" onClick={() => setSyncForm(null)}>取消</Button>
            <Button onClick={handleSyncNode}>下发同步</Button>
          </>
        )}
      >
        {syncForm && (
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <label className="text-sm font-medium">HTTP 端口</label>
                <Input type="number" value={syncForm.httpPort} onChange={(event) => setSyncForm({ ...syncForm, httpPort: Number(event.target.value) })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">SOCKS5 端口</label>
                <Input type="number" value={syncForm.socksPort} onChange={(event) => setSyncForm({ ...syncForm, socksPort: Number(event.target.value) })} />
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">GOST 版本</label>
              <Input value={syncForm.gostVersion} onChange={(event) => setSyncForm({ ...syncForm, gostVersion: event.target.value })} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">出口网络</label>
              <select
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                value={syncForm.egressMode}
                onChange={(event) => setSyncForm({ ...syncForm, egressMode: event.target.value })}
              >
                <option value="auto">自动</option>
                <option value="ipv4">强制 IPv4</option>
                <option value="prefer_ipv6">IPv6 优先（失败自动重试 IPv4）</option>
                <option value="ipv6">强制 IPv6</option>
                <option value="custom">自定义接口或本机 IP</option>
              </select>
            </div>
            {syncForm.egressMode === 'custom' && (
              <div className="space-y-2">
                <label className="text-sm font-medium">自定义接口/IP</label>
                <Input placeholder="eth0 或 2a03:..." value={syncForm.egressInterface} onChange={(event) => setSyncForm({ ...syncForm, egressInterface: event.target.value })} />
              </div>
            )}
          </div>
        )}
      </Modal>
    </motion.div>
  );
}
