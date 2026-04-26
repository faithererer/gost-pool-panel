import * as React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AppProvider, useAppContext } from './api/AppContext';
import Login from './pages/Login';
import Layout from './pages/Layout';
import Dashboard from './pages/Dashboard';
import Nodes from './pages/Nodes';
import Tokens from './pages/Tokens';
import Groups from './pages/Groups';
import Pools from './pages/Pools';
import Tasks from './pages/Tasks';
import Settings from './pages/Settings';

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { state, loading } = useAppContext();

  if (loading) {
    return <div className="flex h-screen items-center justify-center bg-background"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div></div>;
  }

  if (!state) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

function App() {
  return (
    <AppProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="nodes" element={<Nodes />} />
            <Route path="tokens" element={<Tokens />} />
            <Route path="groups" element={<Groups />} />
            <Route path="pools" element={<Pools />} />
            <Route path="tasks" element={<Tasks />} />
            <Route path="settings" element={<Settings />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AppProvider>
  );
}

export default App;