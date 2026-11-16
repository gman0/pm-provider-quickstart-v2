import { useState } from 'react';
import { AuthProvider, useAuth } from './auth';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';

export default function App() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  );
}

function Shell() {
  const { me, loading } = useAuth();
  const [mode, setMode] = useState<'login' | 'register'>('login');

  if (loading) {
    return <div className="p-8 text-sm text-neutral-500">Loading…</div>;
  }
  if (!me) {
    return mode === 'login' ? (
      <Login onDone={() => {}} onSwitch={() => setMode('register')} />
    ) : (
      <Register onDone={() => {}} onSwitch={() => setMode('login')} />
    );
  }
  return <Dashboard />;
}
