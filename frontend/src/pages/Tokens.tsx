import { useState } from 'react';
import { useAppContext } from '../api/AppContext';
import { Card, CardContent } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { KeyRound, Copy, Check, Clock, Info, Trash2 } from 'lucide-react';
import { Input } from '../components/ui/input';
import { motion } from 'framer-motion';
import { api } from '../api';
import { Modal } from '../components/ui/modal';
import { ConfirmDialog } from '../components/ui/confirm-dialog';
import { RegisterToken } from '../api/types';
import { copyText } from '../lib/clipboard';

export default function Tokens() {
  const { state, refreshState, notify } = useAppContext();
  const [copiedToken, setCopiedToken] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [newTokenName, setNewTokenName] = useState('');
  const [newTokenTtl, setNewTokenTtl] = useState(24);
  const [deletingToken, setDeletingToken] = useState<RegisterToken | null>(null);

  if (!state) return null;

  const handleCopy = async (command: string, id: string) => {
    const ok = await copyText(command);
    if (ok) {
      setCopiedToken(id);
      setTimeout(() => setCopiedToken(null), 2000);
      notify({ type: 'success', title: '安装命令已复制' });
      return;
    }
    notify({ type: 'error', title: '复制失败', message: '浏览器限制了剪贴板访问，请手动选中命令复制。' });
  };

  const handleCreate = async () => {
    try {
      await api.post('/register-tokens', { name: newTokenName, ttlHours: newTokenTtl });
      await refreshState();
      setIsCreating(false);
      setNewTokenName('');
      setNewTokenTtl(24);
      notify({ type: 'success', title: '令牌已生成', message: '复制一键安装命令到 Linux VPS 执行即可接入。' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '创建失败', message: error.response?.data?.error || '请稍后重试。' });
    }
  };

  const handleDelete = async () => {
    if (!deletingToken) return;
    try {
      await api.delete(`/register-tokens/${encodeURIComponent(deletingToken.token)}`);
      await refreshState();
      notify({ type: 'success', title: '令牌记录已删除' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '删除失败', message: error.response?.data?.error || '请稍后重试。' });
    } finally {
      setDeletingToken(null);
    }
  };

  const tokens = [...state.registerTokens].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold tracking-tight">接入令牌</h1>
        <Button onClick={() => setIsCreating(true)}>生成令牌</Button>
      </div>

      <div className="bg-primary/5 text-primary text-sm p-4 rounded-md flex items-start gap-3 border border-primary/20">
        <Info className="h-5 w-5 shrink-0 mt-0.5" />
        <p>令牌是一次性的，用于新节点安装并注册到面板。现有节点如果在同一主机上重装，可以通过保留身份信息进行原地升级，不需要新的令牌。</p>
      </div>

      <div className="grid gap-4">
        {tokens.map((token, i) => {
          const isExpired = new Date(token.expiresAt).getTime() < Date.now();
          return (
            <motion.div
              key={token.token}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.2, delay: i * 0.05 }}
            >
              <Card className={`border-l-4 ${token.used ? 'border-l-muted' : isExpired ? 'border-l-destructive' : 'border-l-green-500'}`}>
                <CardContent className="p-6 flex flex-col md:flex-row md:items-center justify-between gap-4">
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <KeyRound className="h-5 w-5 text-muted-foreground" />
                      <h3 className="font-semibold text-lg">{token.name}</h3>
                      <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${token.used ? 'bg-muted text-muted-foreground' : isExpired ? 'bg-destructive/10 text-destructive' : 'bg-green-500/10 text-green-500'}`}>
                        {token.used ? '已使用' : isExpired ? '已过期' : '可用'}
                      </span>
                    </div>
                    <div className="text-xs text-muted-foreground flex items-center gap-2">
                      <Clock className="h-4 w-4" />
                      创建时间: {new Date(token.createdAt).toLocaleString()} |
                      过期时间: {new Date(token.expiresAt).toLocaleString()}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 w-full md:w-auto">
                    <Input
                      readOnly
                      value={token.installCommand}
                      className="font-mono text-xs bg-muted border-transparent flex-1 md:w-80"
                    />
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={() => handleCopy(token.installCommand, token.token)}
                      title="复制安装命令"
                    >
                      {copiedToken === token.token ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                      onClick={() => setDeletingToken(token)}
                      title="删除令牌记录"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          );
        })}
      </div>

      {tokens.length === 0 && (
        <div className="text-center py-20 text-muted-foreground">
          <KeyRound className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>暂无令牌，点击右上角生成一个。</p>
        </div>
      )}

      <Modal
        isOpen={isCreating}
        onClose={() => setIsCreating(false)}
        title="生成新令牌"
        footer={
          <>
            <Button variant="outline" onClick={() => setIsCreating(false)}>取消</Button>
            <Button onClick={handleCreate} disabled={!newTokenName}>生成</Button>
          </>
        }
      >
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">节点名称前缀 / 标识</label>
            <Input
              placeholder="例如: hk-01"
              value={newTokenName}
              onChange={(e) => setNewTokenName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">有效期 (小时)</label>
            <Input
              type="number"
              min={1}
              max={720}
              value={newTokenTtl}
              onChange={(e) => setNewTokenTtl(Number(e.target.value))}
            />
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deletingToken}
        onClose={() => setDeletingToken(null)}
        title="确认删除令牌记录？"
        description={`将删除 ${deletingToken?.name || deletingToken?.token} 的接入令牌记录。已接入节点不会受影响。`}
        confirmText="删除"
        onConfirm={handleDelete}
      />

    </motion.div>
  );
}
