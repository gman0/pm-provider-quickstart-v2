import { useState } from 'react';
import { useAuth } from '../auth';
import Tokens from './Tokens';
import Accesses from './Accesses';
import Profile from './Profile';

type Tab = 'tokens' | 'accesses' | 'profile' | 'settings';

export default function Dashboard() {
  const { me, logout, deleteAccount } = useAuth();
  const [tab, setTab] = useState<Tab>('tokens');
  if (!me) return null;

  return (
    <div className="mx-auto max-w-4xl p-8">
      <header className="flex items-center justify-between border-b border-neutral-200 pb-4">
        <h1 className="text-xl font-semibold">Wildwest Portal</h1>
        <div className="text-sm text-neutral-600">{me.email}</div>
      </header>

      <nav className="mt-4 flex gap-6 text-sm">
        <TabBtn label="Tokens" active={tab === 'tokens'} onClick={() => setTab('tokens')} />
        <TabBtn label="Accesses" active={tab === 'accesses'} onClick={() => setTab('accesses')} />
        <TabBtn label="Profile" active={tab === 'profile'} onClick={() => setTab('profile')} />
        <TabBtn label="Settings" active={tab === 'settings'} onClick={() => setTab('settings')} />
      </nav>

      <main className="mt-6">
        {tab === 'tokens' && <Tokens />}
        {tab === 'accesses' && <Accesses />}
        {tab === 'profile' && <Profile />}
        {tab === 'settings' && (
          <section>
            <h2 className="text-lg font-semibold">Settings</h2>
            <div className="mt-4 flex gap-3">
              <button
                onClick={logout}
                className="rounded border border-neutral-300 bg-white px-3 py-2 text-sm hover:bg-neutral-50"
              >
                Log out
              </button>
              <button
                onClick={() => {
                  if (confirm('Delete your account? All tokens and Accesses will be revoked.')) deleteAccount();
                }}
                className="rounded border border-red-300 bg-white px-3 py-2 text-sm text-red-700 hover:bg-red-50"
              >
                Delete account
              </button>
            </div>
          </section>
        )}
      </main>
    </div>
  );
}

function TabBtn({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={
        active
          ? 'border-b-2 border-neutral-900 pb-1 font-medium text-neutral-900'
          : 'pb-1 text-neutral-500 hover:text-neutral-900'
      }
    >
      {label}
    </button>
  );
}
