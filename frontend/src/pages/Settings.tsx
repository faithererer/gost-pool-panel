import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { AlertTriangle, Eye, EyeOff, Shield } from 'lucide-react';
import { api } from '../api';
import { useAppContext } from '../api/AppContext';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '../components/ui/card';
import { ConfirmDialog } from '../components/ui/confirm-dialog';
import { Input } from '../components/ui/input';

export default function Settings() {
  const { state, refreshState, notify } = useAppContext();
  const [proxyUsername, setProxyUsername] = useState('');
  const [proxyPassword, setProxyPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [confirmSave, setConfirmSave] = useState(false);
  const [confirmCleanup, setConfirmCleanup] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (state?.settings) {
      setProxyUsername(state.settings.proxyUsername || '');
      setProxyPassword(state.settings.proxyPassword || '');
    }
  }, [state]);

  if (!state) return null;

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.patch('/settings', { proxyUsername, proxyPassword });
      await refreshState();
      notify({ type: 'success', title: '设置已保存', message: '节点同步任务已创建，Agent 下次轮询会应用新认证。' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '保存失败', message: error.response?.data?.error || '请稍后重试。' });
    } finally {
      setIsSaving(false);
      setConfirmSave(false);
    }
  };

  const handleCleanup = async () => {
    try {
      const response = await api.post('/nodes/cleanup-uninstalled');
      await refreshState();
      notify({ type: 'success', title: '清理完成', message: `已清理 ${response.data?.count || 0} 个已卸载节点。` });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '清理失败', message: error.response?.data?.error || '请稍后重试。' });
    } finally {
      setConfirmCleanup(false);
    }
  };

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">系统设置</h1>
        <p className="mt-1 text-sm text-muted-foreground">配置代理认证、查看版本信息，并执行低频维护操作。</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-primary" /> 全局代理认证
          </CardTitle>
          <CardDescription>所有节点代理和代理池入口使用同一组代理用户名和密码。保存后会自动下发同步任务。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">代理用户名</label>
            <Input
              type="text"
              placeholder="proxy"
              value={proxyUsername}
              onChange={(event) => setProxyUsername(event.target.value)}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">代理密码</label>
            <div className="relative">
              <Input
                type={showPassword ? 'text' : 'password'}
                placeholder="请输入代理密码"
                value={proxyPassword}
                onChange={(event) => setProxyPassword(event.target.value)}
                className="pr-12"
              />
              <button
                type="button"
                className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? '隐藏密码' : '显示密码'}
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
        </CardContent>
        <CardFooter className="flex justify-end border-t bg-muted/50 py-4">
          <Button onClick={() => setConfirmSave(true)} disabled={isSaving || !proxyUsername || !proxyPassword}>
            保存并同步
          </Button>
        </CardFooter>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>运行信息</CardTitle>
          <CardDescription>这些值来自当前面板进程和配置。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm sm:grid-cols-2">
          <div className="rounded-md border p-3">
            <div className="text-muted-foreground">面板访问地址</div>
            <div className="mt-1 break-all font-mono">{state.baseURL || '-'}</div>
          </div>
          <div className="rounded-md border p-3">
            <div className="text-muted-foreground">版本</div>
            <div className="mt-1 font-mono">panel {state.versions.panel} / agent {state.versions.agent}</div>
          </div>
        </CardContent>
      </Card>

      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-destructive">
            <AlertTriangle className="h-5 w-5" /> 维护操作
          </CardTitle>
          <CardDescription>这些操作会修改面板中的持久化记录，执行前请确认影响范围。</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4 rounded-md border p-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h4 className="text-sm font-medium">清理已卸载节点</h4>
              <p className="mt-1 text-xs text-muted-foreground">删除状态为 agent uninstalled 的节点记录，并清理它们的关联任务。</p>
            </div>
            <Button variant="destructive" size="sm" onClick={() => setConfirmCleanup(true)}>
              清理记录
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        isOpen={confirmSave}
        onClose={() => setConfirmSave(false)}
        title="确认更新全局代理认证？"
        description="保存后会为所有未卸载节点创建同步任务，并重启已启用的代理池入口。短时间内现有代理连接可能中断。"
        confirmText="保存并同步"
        onConfirm={handleSave}
      />

      <ConfirmDialog
        isOpen={confirmCleanup}
        onClose={() => setConfirmCleanup(false)}
        title="确认清理已卸载节点？"
        description="该操作会删除面板中的已卸载节点记录及关联任务，不会连接或操作 VPS。"
        confirmText="清理"
        onConfirm={handleCleanup}
      />
    </motion.div>
  );
}
