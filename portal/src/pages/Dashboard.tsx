import { useAuth } from '../auth';

export default function Dashboard() {
  const { me, logout, deleteAccount } = useAuth();
  if (!me) return null;

  return (
    <div className="mx-auto max-w-2xl p-8">
      <header className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Wildwest Portal</h1>
        <div className="text-sm text-neutral-600">{me.email}</div>
      </header>

      <section className="mt-8 space-y-4">
        <div className="rounded border border-neutral-200 bg-white p-4 shadow-sm">
          <div className="text-sm font-medium">Tokens</div>
          <div className="mt-1 text-sm text-neutral-500">Coming in milestone 3.</div>
        </div>
        <div className="rounded border border-neutral-200 bg-white p-4 shadow-sm">
          <div className="text-sm font-medium">Accesses</div>
          <div className="mt-1 text-sm text-neutral-500">Coming in milestone 4.</div>
        </div>
      </section>

      <section className="mt-8 flex gap-3">
        <button
          onClick={logout}
          className="rounded border border-neutral-300 bg-white px-3 py-2 text-sm hover:bg-neutral-50"
        >
          Log out
        </button>
        <button
          onClick={() => {
            if (confirm('Delete your account? This cannot be undone.')) deleteAccount();
          }}
          className="rounded border border-red-300 bg-white px-3 py-2 text-sm text-red-700 hover:bg-red-50"
        >
          Delete account
        </button>
      </section>
    </div>
  );
}
