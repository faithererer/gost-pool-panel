import { useState } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard,
  Server,
  KeyRound,
  FolderTree,
  Network,
  ListTodo,
  Settings,
  LogOut,
  Menu,
  ShieldAlert
} from 'lucide-react';
import { Button } from '../components/ui/button';
import { api } from '../api';
import { useAppContext } from '../api/AppContext';
import { motion, AnimatePresence } from 'framer-motion';
import { ToastViewport } from '../components/ui/toast';

export default function Layout() {
  const { state, notices, dismissNotice } = useAppContext();
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const handleLogout = async () => {
    try {
      await api.post('/logout');
      navigate('/login');
    } catch (e) {
      console.error(e);
    }
  };

  const navItems = [
    { to: '/', label: '概览', icon: LayoutDashboard },
    { to: '/nodes', label: '节点', icon: Server },
    { to: '/tokens', label: '接入令牌', icon: KeyRound },
    { to: '/groups', label: '分组', icon: FolderTree },
    { to: '/pools', label: '代理池', icon: Network },
    { to: '/tasks', label: '任务', icon: ListTodo },
    { to: '/settings', label: '设置', icon: Settings },
  ];

  return (
    <div className="flex h-screen w-full bg-background dark text-foreground overflow-hidden">
      <ToastViewport notices={notices} onDismiss={dismissNotice} />
      {/* Sidebar */}
      <AnimatePresence initial={false}>
        {sidebarOpen && (
          <motion.aside
            initial={{ width: 0, opacity: 0 }}
            animate={{ width: 256, opacity: 1 }}
            exit={{ width: 0, opacity: 0 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="h-full bg-card border-r border-border flex flex-col shrink-0 overflow-hidden"
          >
            <div className="h-16 flex items-center px-6 border-b border-border shrink-0">
              <ShieldAlert className="h-6 w-6 text-primary mr-3" />
              <span className="font-bold text-lg tracking-tight whitespace-nowrap">GOST Pool</span>
            </div>

            <nav className="flex-1 overflow-y-auto py-6 px-3 space-y-1">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) => `
                    flex items-center px-3 py-2.5 rounded-md text-sm font-medium transition-colors
                    ${isActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                    }
                  `}
                >
                  <item.icon className="h-5 w-5 mr-3 shrink-0" />
                  {item.label}
                </NavLink>
              ))}
            </nav>

            <div className="p-4 border-t border-border shrink-0">
              <div className="flex items-center justify-between text-sm text-muted-foreground mb-4 px-2">
                <span className="truncate">总计 {state?.summary.totalNodes} 节点</span>
                <span className="w-2 h-2 rounded-full bg-green-500"></span>
              </div>
              <Button variant="outline" className="w-full justify-start text-muted-foreground hover:text-foreground" onClick={handleLogout}>
                <LogOut className="h-4 w-4 mr-2" />
                退出登录
              </Button>
            </div>
          </motion.aside>
        )}
      </AnimatePresence>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        <header className="h-16 flex items-center px-4 md:px-6 border-b border-border bg-card shrink-0">
          <Button variant="ghost" size="icon" onClick={() => setSidebarOpen(!sidebarOpen)} className="mr-4 text-muted-foreground">
            <Menu className="h-5 w-5" />
          </Button>
          <div className="ml-auto flex items-center space-x-4">
            {state?.summary.failedTasks ? (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-destructive/10 text-destructive">
                {state.summary.failedTasks} 个失败任务
              </span>
            ) : null}
            <span className="text-sm font-medium text-muted-foreground">
              v{state?.versions.panel}
            </span>
          </div>
        </header>

        <main className="flex-1 overflow-y-auto p-4 md:p-6 bg-background relative">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
