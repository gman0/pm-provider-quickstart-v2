import { useEffect, useState } from 'react';
import { AuthProvider, useAuth } from './auth';
import Header from './components/Header';
import Welcome from './pages/Welcome';
import Login from './pages/Login';
import Register from './pages/Register';
import Admin from './pages/Admin';

type OutMode = 'login' | 'register';
type InMode = 'welcome' | 'admin';

export default function App() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  );
}

function Shell() {
  const { me, loading } = useAuth();
  const [outMode, setOutMode] = useState<OutMode>('login');
  const [inMode, setInMode] = useState<InMode>('welcome');

  useEffect(() => {
    if (!me) setInMode('welcome');
  }, [me]);

  if (loading) {
    return <div className="p-8 text-sm text-neutral-500">Loading…</div>;
  }

  if (me) {
    return inMode === 'welcome' ? (
      <Welcome onOpenAdmin={() => setInMode('admin')} />
    ) : (
      <Admin onBack={() => setInMode('welcome')} />
    );
  }

  return (
    <div className="min-h-screen bg-neutral-50">
      <Header />
      <div className="py-6">
        {outMode === 'login' ? (
          <Login onDone={() => {}} onSwitch={() => setOutMode('register')} />
        ) : (
          <Register onDone={() => {}} onSwitch={() => setOutMode('login')} />
        )}
      </div>
    </div>
  );
}
