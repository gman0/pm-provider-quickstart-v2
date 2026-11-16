import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api';

interface TokenView {
  identity: string;
  note: string;
  tokenID: string;
  createdAt: string;
  accessCount: number;
}

interface CreateTokenResponse {
  identity: string;
  note: string;
  accessToken: string;
  bundle: string;
}

export default function Tokens() {
  const qc = useQueryClient();
  const tokensQ = useQuery({
    queryKey: ['tokens'],
    queryFn: () => api<TokenView[]>('/api/tokens'),
  });
  const [note, setNote] = useState('');
  const [created, setCreated] = useState<CreateTokenResponse | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const createMut = useMutation({
    mutationFn: (n: string) =>
      api<CreateTokenResponse>('/api/tokens', {
        method: 'POST',
        body: JSON.stringify({ note: n }),
      }),
    onSuccess: (data) => {
      setCreated(data);
      setNote('');
      qc.invalidateQueries({ queryKey: ['tokens'] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: (identity: string) => api(`/api/tokens/${identity}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tokens'] }),
  });

  return (
    <section>
      <h2 className="text-lg font-semibold">Access tokens</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Each token is an <code>AccessIdentity</code>. Mint one here, apply the
        generated bundle in your Platform Mesh workspace, and the binding
        shows up under Accesses.
      </p>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          setErr(null);
          createMut.mutate(note);
        }}
        className="mt-4 flex gap-2"
      >
        <input
          placeholder="Optional note (e.g. laptop dev cluster)"
          value={note}
          onChange={(e) => setNote(e.target.value)}
          className="flex-1 rounded border border-neutral-300 px-3 py-2 text-sm"
        />
        <button
          type="submit"
          disabled={createMut.isPending}
          className="rounded bg-neutral-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {createMut.isPending ? 'Minting…' : 'Mint token'}
        </button>
      </form>
      {err && <div className="mt-2 text-sm text-red-600">{err}</div>}

      {created && (
        <div className="mt-4 rounded border border-emerald-300 bg-emerald-50 p-4">
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium">New token minted</div>
            <button
              onClick={() => setCreated(null)}
              className="text-xs text-neutral-600 hover:text-neutral-900"
            >
              Dismiss
            </button>
          </div>
          <div className="mt-3 text-xs font-medium text-neutral-700">Access token</div>
          <pre className="mt-1 overflow-x-auto rounded bg-white p-2 text-xs">{created.accessToken}</pre>
          <button
            onClick={() => navigator.clipboard.writeText(created.accessToken)}
            className="mt-1 text-xs text-neutral-600 hover:text-neutral-900"
          >
            Copy token
          </button>

          <div className="mt-3 text-xs font-medium text-neutral-700">
            YAML bundle (apply in your PM workspace)
          </div>
          <pre className="mt-1 max-h-72 overflow-auto rounded bg-white p-2 text-xs">{created.bundle}</pre>
          <button
            onClick={() => navigator.clipboard.writeText(created.bundle)}
            className="mt-1 text-xs text-neutral-600 hover:text-neutral-900"
          >
            Copy bundle
          </button>
        </div>
      )}

      <div className="mt-6">
        {tokensQ.isLoading && <div className="text-sm text-neutral-500">Loading…</div>}
        {tokensQ.isError && (
          <div className="text-sm text-red-600">Failed to load: {String(tokensQ.error)}</div>
        )}
        {tokensQ.data && tokensQ.data.length === 0 && (
          <div className="text-sm text-neutral-500">No tokens yet.</div>
        )}
        {tokensQ.data && tokensQ.data.length > 0 && (
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-neutral-200 text-left text-neutral-600">
                <th className="py-2 pr-4 font-medium">Note / Identity</th>
                <th className="py-2 pr-4 font-medium">Token ID</th>
                <th className="py-2 pr-4 font-medium">Accesses</th>
                <th className="py-2 pr-4 font-medium">Created</th>
                <th className="py-2"></th>
              </tr>
            </thead>
            <tbody>
              {tokensQ.data.map((t) => (
                <tr key={t.identity} className="border-b border-neutral-100">
                  <td className="py-2 pr-4">
                    <div className="font-medium">{t.note || <span className="text-neutral-500">(no note)</span>}</div>
                    <div className="text-xs text-neutral-500">{t.identity}</div>
                  </td>
                  <td className="py-2 pr-4 font-mono text-xs">{t.tokenID || '—'}</td>
                  <td className="py-2 pr-4">{t.accessCount}</td>
                  <td className="py-2 pr-4 text-xs text-neutral-600">
                    {new Date(t.createdAt).toLocaleString()}
                  </td>
                  <td className="py-2 text-right">
                    <button
                      onClick={() => {
                        if (confirm(`Revoke token "${t.note || t.identity}"? Existing Accesses keep working until deleted individually.`)) {
                          deleteMut.mutate(t.identity);
                        }
                      }}
                      className="text-xs text-red-700 hover:underline"
                    >
                      Revoke
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
