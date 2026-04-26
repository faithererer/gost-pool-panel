import React, { useState } from 'react';
import { api } from '../api';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '../components/ui/card';
import { useNavigate } from 'react-router-dom';
import { Shield } from 'lucide-react';
import { motion } from 'framer-motion';
import { useAppContext } from '../api/AppContext';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { refreshState } = useAppContext();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await api.post('/login', { username, password });
      await refreshState();
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-screen w-full items-center justify-center bg-background p-4 dark">
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
        className="w-full max-w-md"
      >
        <Card className="border-muted bg-card text-card-foreground shadow-xl">
          <CardHeader className="space-y-1 items-center pb-8 pt-10">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
              <Shield className="h-8 w-8 text-primary" />
            </div>
            <CardTitle className="text-2xl font-bold tracking-tight">GOST Pool Panel</CardTitle>
            <CardDescription className="text-muted-foreground text-center px-4 pt-2">
              登录后管理节点、代理池和远程任务。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <Input
                  type="text"
                  placeholder="用户名"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  disabled={loading}
                  className="h-12 bg-background"
                />
              </div>
              <div className="space-y-2">
                <Input
                  type="password"
                  placeholder="密码"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={loading}
                  className="h-12 bg-background"
                />
              </div>
              {error && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  className="text-sm text-destructive mt-2"
                >
                  {error}
                </motion.div>
              )}
              <Button type="submit" className="w-full h-12 text-base font-semibold" disabled={loading}>
                {loading ? '正在登录...' : '登录'}
              </Button>
            </form>
          </CardContent>
          <CardFooter className="flex justify-center pb-8 text-xs text-muted-foreground">
            &copy; {new Date().getFullYear()} GOST Pool
          </CardFooter>
        </Card>
      </motion.div>
    </div>
  );
}