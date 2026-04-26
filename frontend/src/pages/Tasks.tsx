import { useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import { CheckCircle2, Clock, Filter, ListTodo, PlayCircle, XCircle } from 'lucide-react';
import { api } from '../api';
import { useAppContext } from '../api/AppContext';
import { Task, TaskStatus } from '../api/types';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Modal } from '../components/ui/modal';

const filters: Array<{ value: 'all' | TaskStatus; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'failed', label: '失败' },
  { value: 'pending', label: '排队中' },
  { value: 'running', label: '运行中' },
  { value: 'success', label: '成功' },
];

function statusLabel(status: string) {
  switch (status) {
    case 'success':
      return '成功';
    case 'failed':
      return '失败';
    case 'running':
      return '运行中';
    case 'pending':
      return '排队中';
    default:
      return status || '未知';
  }
}

function statusIcon(status: string) {
  switch (status) {
    case 'success':
      return <CheckCircle2 className="h-5 w-5 text-green-500" />;
    case 'failed':
      return <XCircle className="h-5 w-5 text-destructive" />;
    case 'running':
      return <PlayCircle className="h-5 w-5 text-indigo-500" />;
    case 'pending':
      return <Clock className="h-5 w-5 text-yellow-500" />;
    default:
      return <ListTodo className="h-5 w-5 text-muted-foreground" />;
  }
}

function formatDate(value: string) {
  if (!value || value === '0001-01-01T00:00:00Z') return '-';
  return new Date(value).toLocaleString();
}

function formatPayload(payload: string) {
  if (!payload) return '{}';
  try {
    return JSON.stringify(JSON.parse(payload), null, 2);
  } catch {
    return payload;
  }
}

export default function Tasks() {
  const { state, refreshState } = useAppContext();
  const [filter, setFilter] = useState<'all' | TaskStatus>('all');
  const [search, setSearch] = useState('');
  const [viewingTask, setViewingTask] = useState<Task | null>(null);

  const nodeNames = useMemo(() => {
    const names = new Map<string, string>();
    state?.nodes.forEach((node) => names.set(node.id, node.name));
    return names;
  }, [state?.nodes]);

  if (!state) return null;

  const handleRetry = async (taskId: string) => {
    try {
      await api.post(`/tasks/${taskId}/retry`);
      await refreshState();
    } catch (error: any) {
      console.error(error);
      alert(error.response?.data?.error || '重试失败');
    }
  };

  const filteredTasks = state.tasks.filter((task) => {
    if (filter !== 'all' && task.status !== filter) return false;
    const keyword = search.trim().toLowerCase();
    if (!keyword) return true;
    return [task.nodeId, nodeNames.get(task.nodeId), task.type, task.error]
      .some((value) => (value || '').toLowerCase().includes(keyword));
  });

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">任务管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">查看节点远程任务的排队、执行、失败和重试状态。</p>
        </div>

        <div className="flex flex-wrap gap-2">
          {filters.map((item) => (
            <Button
              key={item.value}
              variant={filter === item.value ? 'default' : 'outline'}
              size="sm"
              className={filter === 'failed' && item.value === 'failed' ? 'bg-destructive hover:bg-destructive/90' : ''}
              onClick={() => setFilter(item.value)}
            >
              {item.label}
            </Button>
          ))}
        </div>
      </div>

      <div className="flex w-full max-w-sm items-center gap-2">
        <Filter className="h-5 w-5 text-muted-foreground" />
        <Input type="search" placeholder="搜索节点、任务类型或错误信息" value={search} onChange={(event) => setSearch(event.target.value)} />
      </div>

      <div className="overflow-hidden rounded-md border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-muted/50 text-muted-foreground">
              <tr>
                <th className="px-4 py-3 font-medium">状态</th>
                <th className="px-4 py-3 font-medium">任务</th>
                <th className="px-4 py-3 font-medium">节点</th>
                <th className="px-4 py-3 font-medium">创建时间</th>
                <th className="px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {filteredTasks.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-muted-foreground">
                    没有找到匹配的任务。
                  </td>
                </tr>
              ) : filteredTasks.map((task) => (
                <tr key={task.id} className={`border-t border-border transition-colors hover:bg-muted/50 ${task.status === 'failed' ? 'bg-destructive/5' : ''}`}>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {statusIcon(task.status)}
                      <span>{statusLabel(task.status)}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 font-medium">{task.type}</td>
                  <td className="px-4 py-3">
                    <div>{nodeNames.get(task.nodeId) || '未知节点'}</div>
                    <div className="font-mono text-xs text-muted-foreground">{task.nodeId}</div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{formatDate(task.createdAt)}</td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      {task.status === 'failed' && (
                        <Button variant="outline" size="sm" onClick={() => handleRetry(task.id)}>重试</Button>
                      )}
                      <Button variant="ghost" size="sm" onClick={() => setViewingTask(task)}>查看详情</Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Modal
        isOpen={!!viewingTask}
        onClose={() => setViewingTask(null)}
        title="任务详情"
        footer={<Button onClick={() => setViewingTask(null)}>关闭</Button>}
      >
        {viewingTask && (
          <div className="max-h-[60vh] space-y-4 overflow-y-auto py-2 pr-2">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="mb-1 block text-muted-foreground">任务 ID</span>
                <span className="font-mono text-xs">{viewingTask.id}</span>
              </div>
              <div>
                <span className="mb-1 block text-muted-foreground">节点</span>
                <span>{nodeNames.get(viewingTask.nodeId) || viewingTask.nodeId}</span>
              </div>
              <div>
                <span className="mb-1 block text-muted-foreground">类型</span>
                <span>{viewingTask.type}</span>
              </div>
              <div>
                <span className="mb-1 block text-muted-foreground">状态</span>
                <span className="inline-flex items-center gap-1">{statusIcon(viewingTask.status)} {statusLabel(viewingTask.status)}</span>
              </div>
            </div>

            <div className="space-y-2">
              <span className="text-sm text-muted-foreground">Payload</span>
              <pre className="overflow-x-auto rounded-md border bg-muted p-3 font-mono text-xs">{formatPayload(viewingTask.payload)}</pre>
            </div>

            {viewingTask.result && (
              <div className="space-y-2">
                <span className="text-sm text-muted-foreground">Result</span>
                <pre className="overflow-x-auto rounded-md border bg-muted p-3 font-mono text-xs">{viewingTask.result}</pre>
              </div>
            )}

            {viewingTask.error && (
              <div className="space-y-2">
                <span className="flex items-center gap-1 text-sm font-medium text-destructive">
                  <XCircle className="h-4 w-4" /> 错误信息
                </span>
                <pre className="whitespace-pre-wrap rounded-md border border-destructive/20 bg-destructive/10 p-3 font-mono text-xs text-destructive">{viewingTask.error}</pre>
              </div>
            )}

            <div className="grid grid-cols-3 gap-2 border-t pt-4 text-xs text-muted-foreground">
              <div>创建：{formatDate(viewingTask.createdAt)}</div>
              <div>开始：{formatDate(viewingTask.startedAt)}</div>
              <div>完成：{formatDate(viewingTask.finishedAt)}</div>
            </div>
          </div>
        )}
      </Modal>
    </motion.div>
  );
}