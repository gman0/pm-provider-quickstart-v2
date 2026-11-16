import { useState } from 'react';
import { useAuth } from '../auth';
import Header from '../components/Header';
import Tokens from './Tokens';
import Accesses from './Accesses';
import Profile from './Profile';

type Tab = 'tokens' | 'accesses' | 'profile' | 'settings';

export default function Admin({ onBack }: { onBack: () => void }) {
  const { me, logout, deleteAccount } = useAuth();
  const [tab, setTab] = useState<Tab>('tokens');
  if (!me) return null;

  return (
    <div className="min-h-screen bg-neutral-50">
      <Header
        right={
          <div className="flex items-center gap-4">
            <span>{me.email}</span>
            <button onClick={logout} className="text-neutral-900 underline hover:no-underline">
              Log out
            </button>
          </div>
        }
      />

      <div className="mx-auto max-w-5xl px-6 py-6">
        <button
          onClick={onBack}
          className="text-sm text-neutral-600 hover:text-neutral-900"
        >
          ← Home
        </button>

        <nav className="mt-4 flex gap-6 border-b border-neutral-200 text-sm">
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
                    if (confirm('Delete your account? All tokens and Accesses will be revoked.'))
                      deleteAccount();
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
    </div>
  );
}

function TabBtn({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={
        active
          ? 'border-b-2 border-neutral-900 pb-2 font-medium text-neutral-900'
          : 'pb-2 text-neutral-500 hover:text-neutral-900'
      }
    >
      {label}
    </button>
  );
}
