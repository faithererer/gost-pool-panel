import { useState } from 'react';
import { useAppContext } from '../api/AppContext';
import { Card, CardContent } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { FolderTree, Edit, Trash2, Server } from 'lucide-react';
import { Input } from '../components/ui/input';
import { motion } from 'framer-motion';
import { api } from '../api';
import { ConfirmDialog } from '../components/ui/confirm-dialog';
import { Modal } from '../components/ui/modal';

export default function Groups() {
  const { state, refreshState, notify } = useAppContext();
  const [isCreating, setIsCreating] = useState(false);
  const [editingGroup, setEditingGroup] = useState<any>(null);
  const [deletingGroup, setDeletingGroup] = useState<any>(null);

  const [name, setName] = useState('');
  const [remark, setRemark] = useState('');

  if (!state) return null;

  const handleCreate = async () => {
    try {
      await api.post('/groups', { name, remark });
      await refreshState();
      setIsCreating(false);
      setName('');
      setRemark('');
      notify({ type: 'success', title: '分组已创建' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '创建失败', message: error.response?.data?.error || '请填写有效的分组名称。' });
    }
  };

  const handleUpdate = async () => {
    if (!editingGroup) return;
    try {
      await api.patch(`/groups/${editingGroup.id}`, { name, remark });
      await refreshState();
      setEditingGroup(null);
      notify({ type: 'success', title: '分组已更新' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '更新失败', message: error.response?.data?.error || '请稍后重试。' });
    }
  };

  const handleDelete = async () => {
    if (!deletingGroup) return;
    try {
      await api.delete(`/groups/${deletingGroup.id}`);
      await refreshState();
      notify({ type: 'success', title: '分组已删除', message: '相关节点和代理池中的引用也已移除。' });
    } catch (error: any) {
      console.error(error);
      notify({ type: 'error', title: '删除失败', message: error.response?.data?.error || '请稍后重试。' });
    }
  };

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold tracking-tight">分组管理</h1>
        <Button onClick={() => { setIsCreating(true); setName(''); setRemark(''); }}>创建分组</Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {state.groups.map((group, i) => {
          const nodeCount = state.nodes.filter(n => n.groupIds?.includes(group.id)).length;
          const poolCount = state.pools.filter(p => p.groupIds?.includes(group.id)).length;

          return (
            <motion.div
              key={group.id}
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.2, delay: i * 0.05 }}
            >
              <Card>
                <CardContent className="p-6">
                  <div className="flex items-start justify-between">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <FolderTree className="h-5 w-5 text-primary" />
                        <h3 className="font-semibold text-lg">{group.name}</h3>
                      </div>
                      {group.remark && <p className="text-sm text-muted-foreground">{group.remark}</p>}
                    </div>
                    <div className="flex gap-1">
                      <Button variant="ghost" size="icon" onClick={() => { setEditingGroup(group); setName(group.name); setRemark(group.remark); }}>
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => setDeletingGroup(group)}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 mt-4 pt-4 border-t border-border text-sm text-muted-foreground">
                    <span className="flex items-center gap-1.5 bg-muted px-2 py-1 rounded-md">
                      <Server className="h-4 w-4" /> {nodeCount} 个节点
                    </span>
                    <span className="flex items-center gap-1.5 bg-muted px-2 py-1 rounded-md">
                      {poolCount} 个代理池引用
                    </span>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          );
        })}
      </div>

      {state.groups.length === 0 && (
        <div className="text-center py-20 text-muted-foreground">
          <FolderTree className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>暂无分组，点击右上角创建一个。</p>
        </div>
      )}

      <Modal
        isOpen={isCreating || !!editingGroup}
        onClose={() => { setIsCreating(false); setEditingGroup(null); }}
        title={editingGroup ? "编辑分组" : "创建分组"}
        footer={
          <>
            <Button variant="outline" onClick={() => { setIsCreating(false); setEditingGroup(null); }}>取消</Button>
            <Button onClick={editingGroup ? handleUpdate : handleCreate} disabled={!name}>保存</Button>
          </>
        }
      >
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">名称</label>
            <Input
              placeholder="例如: US Nodes"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">备注 (可选)</label>
            <Input
              placeholder="组描述"
              value={remark}
              onChange={(e) => setRemark(e.target.value)}
            />
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deletingGroup}
        onClose={() => setDeletingGroup(null)}
        title="确认删除分组？"
        description={`您即将删除分组 ${deletingGroup?.name}。这会从所有关联的节点和代理池中移除该分组的引用。`}
        onConfirm={handleDelete}
      />

    </motion.div>
  );
}
