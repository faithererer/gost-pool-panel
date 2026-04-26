import { useAppContext } from '../api/AppContext';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Activity, ServerCrash, ArrowUpCircle, CheckCircle2, XCircle } from 'lucide-react';
import { motion } from 'framer-motion';

export default function Dashboard() {
  const { state } = useAppContext();

  if (!state) return null;

  const stats = [
    { label: "所有节点", value: state.summary.totalNodes, icon: Activity, color: "text-blue-500" },
    { label: "在线节点", value: state.summary.onlineNodes, icon: CheckCircle2, color: "text-green-500" },
    { label: "GOST 活跃", value: state.summary.gostActiveNodes, icon: Activity, color: "text-indigo-500" },
    { label: "运行中代理池", value: state.summary.runningPools, icon: ArrowUpCircle, color: "text-purple-500" },
    { label: "失败任务", value: state.summary.failedTasks, icon: XCircle, color: "text-red-500" },
    { label: "过期 Agent 节点", value: state.summary.outdatedAgentNodes, icon: ServerCrash, color: "text-yellow-500" },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className="space-y-6"
    >
      <h1 className="text-3xl font-bold tracking-tight">概览</h1>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {stats.map((stat, i) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3, delay: i * 0.05 }}
          >
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {stat.label}
                </CardTitle>
                <stat.icon className={`h-5 w-5 ${stat.color}`} />
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold">{stat.value}</div>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {state.summary.recentErrorTasks.length > 0 && (
        <Card className="border-destructive/50">
          <CardHeader>
            <CardTitle className="text-destructive flex items-center">
              <XCircle className="mr-2 h-5 w-5" /> 最近失败任务
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {state.summary.recentErrorTasks.map((task) => (
                <div key={task.id} className="flex flex-col gap-1 text-sm bg-destructive/10 p-3 rounded-md border border-destructive/20">
                  <span className="font-semibold">{task.type} (节点: {task.nodeId})</span>
                  <span className="text-muted-foreground">{task.error}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </motion.div>
  );
}