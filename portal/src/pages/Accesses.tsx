import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api';

interface AccessView {
  name: string;
  identityName: string;
  note?: string;
  phase: string;
  lastRefreshed?: string;
  kubeconfigSecret?: string;
  createdAt: string;
}

export default function Accesses() {
  const qc = useQueryClient();
  const q = useQuery({
    queryKey: ['accesses'],
    queryFn: () => api<AccessView[]>('/api/accesses'),
    refetchInterval: 5000,
  });
  const deleteMut = useMutation({
    mutationFn: (name: string) => api(`/api/accesses/${name}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accesses'] }),
  });

  return (
    <section>
      <h2 className="text-lg font-semibold">Accesses</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Bindings produced by the handshake. Deleting one tears down that
        binding without revoking the underlying token.
      </p>
      <div className="mt-4">
        {q.isLoading && <div className="text-sm text-neutral-500">Loading…</div>}
        {q.isError && <div className="text-sm text-red-600">Failed to load: {String(q.error)}</div>}
        {q.data && q.data.length === 0 && (
          <div className="text-sm text-neutral-500">No active Accesses yet.</div>
        )}
        {q.data && q.data.length > 0 && (
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-neutral-200 text-left text-neutral-600">
                <th className="py-2 pr-4 font-medium">Access</th>
                <th className="py-2 pr-4 font-medium">Token</th>
                <th className="py-2 pr-4 font-medium">Phase</th>
                <th className="py-2 pr-4 font-medium">Last refresh</th>
                <th className="py-2"></th>
              </tr>
            </thead>
            <tbody>
              {q.data.map((a) => (
                <tr key={a.name} className="border-b border-neutral-100">
                  <td className="py-2 pr-4">
                    <div className="font-medium font-mono text-xs">{a.name}</div>
                    {a.kubeconfigSecret && (
                      <div className="text-xs text-neutral-500">secret: {a.kubeconfigSecret}</div>
                    )}
                  </td>
                  <td className="py-2 pr-4">
                    <div>{a.note || <span className="text-neutral-500">(no note)</span>}</div>
                    <div className="text-xs text-neutral-500 font-mono">{a.identityName}</div>
                  </td>
                  <td className="py-2 pr-4">
                    <span
                      className={
                        a.phase === 'Active'
                          ? 'rounded bg-emerald-100 px-2 py-0.5 text-xs text-emerald-800'
                          : 'rounded bg-neutral-200 px-2 py-0.5 text-xs text-neutral-700'
                      }
                    >
                      {a.phase || '—'}
                    </span>
                  </td>
                  <td className="py-2 pr-4 text-xs text-neutral-600">
                    {a.lastRefreshed ? new Date(a.lastRefreshed).toLocaleString() : '—'}
                  </td>
                  <td className="py-2 text-right">
                    <button
                      onClick={() => {
                        if (confirm(`Delete Access "${a.name}"? The PM side's credential will stop working.`)) {
                          deleteMut.mutate(a.name);
                        }
                      }}
                      className="text-xs text-red-700 hover:underline"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}
