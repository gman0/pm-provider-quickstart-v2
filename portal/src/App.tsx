import { useQuery } from '@tanstack/react-query';

export default function App() {
  const health = useQuery({
    queryKey: ['healthz'],
    queryFn: async () => {
      const r = await fetch('/healthz');
      if (!r.ok) throw new Error(`healthz ${r.status}`);
      return r.text();
    },
  });

  return (
    <main className="mx-auto max-w-2xl p-8">
      <h1 className="text-2xl font-semibold">Wildwest Portal</h1>
      <p className="mt-2 text-sm text-neutral-600">Skeleton — milestone 1.</p>
      <div className="mt-6 rounded border border-neutral-200 bg-white p-4 shadow-sm">
        <div className="text-sm font-medium">Server health</div>
        <div className="mt-1 text-sm">
          {health.isLoading && <span className="text-neutral-500">checking…</span>}
          {health.isError && <span className="text-red-600">error: {String(health.error)}</span>}
          {health.data && <span className="text-emerald-700">ok</span>}
        </div>
      </div>
    </main>
  );
}
