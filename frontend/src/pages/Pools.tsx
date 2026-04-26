import { useState } from 'react';
import { useAppContext } from '../api/AppContext';
import { Card, CardContent } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { Network, Play, Square, Copy, RefreshCw, Trash2, Edit } from 'lucide-react';
import { Input } from '../components/ui/input';
import { motion } from 'framer-motion';
import { api } from '../api';
import { ConfirmDialog } from '../components/ui/confirm-dialog';
import { Modal } from '../components/ui/modal';
import { copyText } from '../lib/clipboard';

export default function Pools() {
  const { state, refreshState, notify } = useAppContext();
  const [isCreating, setIsCreating] = useState(false);
  const [editingPool, setEditingPool] = useState<any>(null);
  const [deletingPool, setDeletingPool] = useState<any>(null);

  const [name, setName] = useState('');
  const [groupIds, setGroupIds] = useState<string[]>([]);
  const [httpPort, setHttpPort] = useState(18080);
  const [socksPort, setSocksPort] = useState(18081);
  const [strategy, setStrategy] = useState('round');
  const [enabled, setEnabled] = useState(true);

  if (!state) return null;

  const handleCreate = async () => {
    try {
      await api.post('/pools', { name, groupIds, httpPort, socksPort, strategy, enabled });
      await refreshState();
      setIsCreating(false);
      resetForm();
      notify({ type: 'success', title: '代理池已创建', message: enabled ? '入口进程会自动尝试启动。' : '当前为禁用状态，需要启用后才会启动入口。' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '创建失败', message: error.response?.data?.error || '请检查端口、分组和代理认证配置。' });
    }
  };

  const handleUpdate = async () => {
    if (!editingPool) return;
    try {
      await api.patch(`/pools/${editingPool.id}`, { name, groupIds, httpPort, socksPort, strategy, enabled });
      await refreshState();
      setEditingPool(null);
      resetForm();
      notify({ type: 'success', title: '代理池已更新', message: '如果代理池启用，入口进程已按新配置重启。' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '更新失败', message: error.response?.data?.error || '请检查端口是否被占用。' });
    }
  };

  const handleDelete = async () => {
    if (!deletingPool) return;
    try {
      await api.delete(`/pools/${deletingPool.id}`);
      await refreshState();
      setDeletingPool(null);
      notify({ type: 'success', title: '代理池已删除' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '删除失败', message: error.response?.data?.error || '请稍后重试。' });
    }
  };

  const handleRestart = async (id: string) => {
    try {
      await api.post(`/pools/${id}/restart`);
      await refreshState();
      notify({ type: 'success', title: '代理池重启已触发' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '重启失败', message: error.response?.data?.error || '请检查面板容器内 gost 是否存在。' });
    }
  };

  const resetForm = () => {
    setName('');
    setGroupIds([]);
    setHttpPort(18080);
    setSocksPort(18081);
    setStrategy('round');
    setEnabled(true);
  };

  const openEdit = (pool: any) => {
    setEditingPool(pool);
    setName(pool.name);
    setGroupIds(pool.groupIds || []);
    setHttpPort(pool.httpPort);
    setSocksPort(pool.socksPort);
    setStrategy(pool.strategy);
    setEnabled(pool.enabled);
  };

  const copyTestCommand = async (port: number, protocol: string) => {
    const host = window.location.hostname;
    const auth = state.settings?.proxyUsername ? `-U '${state.settings.proxyUsername}:${state.settings.proxyPassword || 'PASSWORD'}' ` : '';
    const ip = host.includes(':') ? `[${host}]` : host;
    const command = `curl -x ${protocol}://${ip}:${port} ${auth}https://api64.ipify.org`;
    const ok = await copyText(command);
    notify(ok
      ? { type: 'success', title: `${protocol.toUpperCase()} 测试命令已复制`, message: '可直接在终端粘贴执行。' }
      : { type: 'error', title: '复制失败', message: '浏览器限制了剪贴板访问，请手动复制测试命令。' });
  };

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold tracking-tight">代理池</h1>
        <Button onClick={() => { setIsCreating(true); resetForm(); }}>创建代理池</Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {state.pools.map((pool, i) => (
          <motion.div
            key={pool.id}
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.2, delay: i * 0.05 }}
          >
            <Card className="hover:border-primary/50 transition-colors">
              <CardContent className="p-6">
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <Network className="h-5 w-5 text-primary" />
                      <h3 className="font-semibold text-lg">{pool.name}</h3>
                    </div>
                  </div>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" onClick={() => handleRestart(pool.id)} title="重启 Runtime">
                      <RefreshCw className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => openEdit(pool)} title="编辑">
                      <Edit className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => setDeletingPool(pool)} title="删除">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                <div className="flex items-center gap-2 mt-2 mb-4">
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium flex items-center gap-1 ${pool.enabled ? 'bg-green-500/10 text-green-500' : 'bg-muted text-muted-foreground'}`}>
                    {pool.enabled ? <Play className="h-3 w-3" /> : <Square className="h-3 w-3" />}
                    {pool.enabled ? '已启用' : '已禁用'}
                  </span>
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${pool.runtimeStatus === 'running' ? 'bg-indigo-500/10 text-indigo-500' : pool.runtimeStatus === 'disabled' ? 'bg-muted text-muted-foreground' : 'bg-destructive/10 text-destructive'}`}>
                    Runtime: {pool.runtimeStatus}
                  </span>
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-secondary text-secondary-foreground">
                    策略: {pool.strategy}
                  </span>
                </div>

                {pool.runtimeError && (
                  <div className="bg-destructive/10 text-destructive text-xs p-2 rounded mb-4 break-words">
                    Error: {pool.runtimeError}
                  </div>
                )}

                <div className="grid grid-cols-2 gap-2 text-sm border-t border-border pt-4">
                  <div className="flex flex-col gap-2">
                    <span className="text-muted-foreground">HTTP 端口: <span className="text-foreground font-mono">{pool.httpPort}</span></span>
                    <Button variant="outline" size="sm" className="w-full h-8 text-xs" onClick={() => copyTestCommand(pool.httpPort, 'http')}>
                      <Copy className="h-3 w-3 mr-1" /> 测试 HTTP
                    </Button>
                  </div>
                  <div className="flex flex-col gap-2">
                    <span className="text-muted-foreground">SOCKS5 端口: <span className="text-foreground font-mono">{pool.socksPort}</span></span>
                    <Button variant="outline" size="sm" className="w-full h-8 text-xs" onClick={() => copyTestCommand(pool.socksPort, 'socks5h')}>
                      <Copy className="h-3 w-3 mr-1" /> 测试 SOCKS
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {state.pools.length === 0 && (
        <div className="text-center py-20 text-muted-foreground">
          <Network className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>暂无代理池，点击右上角创建一个。</p>
        </div>
      )}

      <Modal
        isOpen={isCreating || !!editingPool}
        onClose={() => { setIsCreating(false); setEditingPool(null); }}
        title={editingPool ? "编辑代理池" : "创建代理池"}
        footer={
          <>
            <Button variant="outline" onClick={() => { setIsCreating(false); setEditingPool(null); }}>取消</Button>
            <Button onClick={editingPool ? handleUpdate : handleCreate} disabled={!name}>保存</Button>
          </>
        }
      >
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">名称</label>
            <Input placeholder="例如: US Nodes Pool" value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">HTTP 端口</label>
              <Input type="number" value={httpPort} onChange={(e) => setHttpPort(Number(e.target.value))} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">SOCKS 端口</label>
              <Input type="number" value={socksPort} onChange={(e) => setSocksPort(Number(e.target.value))} />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">负载均衡策略</label>
            <select
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              value={strategy}
              onChange={(e) => setStrategy(e.target.value)}
            >
              <option value="round">轮询 (Round Robin)</option>
              <option value="random">随机 (Random)</option>
            </select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">选择分组 (上游节点来源)</label>
            <div className="max-h-32 overflow-y-auto border border-input rounded-md p-2 space-y-1">
              {state.groups.length === 0 && <p className="text-xs text-muted-foreground p-1">没有可用的分组</p>}
              {state.groups.map(g => (
                <label key={g.id} className="flex items-center gap-2 text-sm p-1 hover:bg-muted rounded">
                  <input
                    type="checkbox"
                    className="rounded border-input text-primary focus:ring-primary"
                    checked={groupIds.includes(g.id)}
                    onChange={(e) => {
                      if (e.target.checked) setGroupIds([...groupIds, g.id]);
                      else setGroupIds(groupIds.filter(id => id !== g.id));
                    }}
                  />
                  {g.name}
                </label>
              ))}
            </div>
          </div>

          <div className="flex items-center gap-2 pt-2">
            <input
              type="checkbox"
              id="pool-enabled"
              className="rounded border-input text-primary focus:ring-primary w-4 h-4"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
            />
            <label htmlFor="pool-enabled" className="text-sm font-medium">启用该代理池</label>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deletingPool}
        onClose={() => setDeletingPool(null)}
        title="确认删除代理池？"
        description={`您即将删除代理池 ${deletingPool?.name}。这会停止该池的所有代理流量，并删除其配置。`}
        onConfirm={handleDelete}
      />

    </motion.div>
  );
}
