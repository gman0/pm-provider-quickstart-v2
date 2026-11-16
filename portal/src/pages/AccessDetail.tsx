import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
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

export default function AccessDetail({
  name,
  onClose,
}: {
  name: string;
  onClose: () => void;
}) {
  const [showRaw, setShowRaw] = useState(false);

  const detailQ = useQuery({
    queryKey: ['access', name],
    queryFn: () => api<AccessView>(`/api/accesses/${name}`),
  });
  const rawQ = useQuery({
    queryKey: ['access', name, 'raw'],
    queryFn: () => api<Record<string, unknown>>(`/api/accesses/${name}/raw`),
    enabled: showRaw,
  });

  return (
    <aside className="fixed inset-y-0 right-0 z-10 flex w-[32rem] max-w-full flex-col overflow-hidden border-l border-neutral-200 bg-white shadow-xl">
      <header className="flex items-center justify-between border-b border-neutral-200 px-4 py-3">
        <div>
          <div className="text-xs text-neutral-500">Access</div>
          <div className="font-mono text-sm">{name}</div>
        </div>
        <button onClick={onClose} className="text-sm text-neutral-600 hover:text-neutral-900">
          Close
        </button>
      </header>

      <div className="flex-1 overflow-auto p-4">
        {detailQ.isLoading && <div className="text-sm text-neutral-500">Loading…</div>}
        {detailQ.isError && (
          <div className="text-sm text-red-600">Failed to load: {String(detailQ.error)}</div>
        )}
        {detailQ.data && (
          <dl className="space-y-3 text-sm">
            <Row label="Phase">
              <span
                className={
                  detailQ.data.phase === 'Active'
                    ? 'rounded bg-emerald-100 px-2 py-0.5 text-xs text-emerald-800'
                    : 'rounded bg-neutral-200 px-2 py-0.5 text-xs text-neutral-700'
                }
              >
                {detailQ.data.phase || '—'}
              </span>
            </Row>
            <Row label="Token note">{detailQ.data.note || <em className="text-neutral-500">(no note)</em>}</Row>
            <Row label="Owning identity">
              <code className="text-xs">{detailQ.data.identityName}</code>
            </Row>
            <Row label="Kubeconfig secret">
              <code className="text-xs">{detailQ.data.kubeconfigSecret || '—'}</code>
            </Row>
            <Row label="Created">{new Date(detailQ.data.createdAt).toLocaleString()}</Row>
            <Row label="Last refresh">
              {detailQ.data.lastRefreshed
                ? new Date(detailQ.data.lastRefreshed).toLocaleString()
                : '—'}
            </Row>
          </dl>
        )}

        <div className="mt-6 border-t border-neutral-200 pt-4">
          <button
            onClick={() => setShowRaw((v) => !v)}
            className="text-sm text-neutral-600 hover:text-neutral-900"
          >
            {showRaw ? 'Hide raw JSON' : 'Show raw JSON'}
          </button>
          {showRaw && rawQ.isLoading && (
            <div className="mt-2 text-sm text-neutral-500">Loading…</div>
          )}
          {showRaw && rawQ.data && (
            <pre className="mt-2 max-h-96 overflow-auto rounded bg-neutral-50 p-2 text-xs">
              {JSON.stringify(rawQ.data, null, 2)}
            </pre>
          )}
        </div>
      </div>
    </aside>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[8rem_1fr] gap-2">
      <dt className="text-xs font-medium text-neutral-500">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}
