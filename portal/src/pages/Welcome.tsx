import { useAuth } from '../auth';
import Header from '../components/Header';

export default function Welcome({ onOpenAdmin }: { onOpenAdmin: () => void }) {
  const { me, logout } = useAuth();

  return (
    <div className="min-h-screen bg-neutral-50">
      <Header
        right={
          <div className="flex items-center gap-4">
            <span>{me?.email}</span>
            <button onClick={logout} className="text-neutral-900 underline hover:no-underline">
              Log out
            </button>
          </div>
        }
      />

      <main className="mx-auto max-w-5xl px-6 py-12">
        <section className="text-center">
          <h1 className="text-3xl font-semibold">Best Wildwest in town!</h1>
          <p className="mx-auto mt-4 max-w-2xl text-sm leading-6 text-neutral-600">
            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do
            eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim
            ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut
            aliquip ex ea commodo consequat. Duis aute irure dolor in
            reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla
            pariatur.
          </p>
        </section>

        <section className="mt-12">
          <h2 className="text-sm font-medium uppercase tracking-wide text-neutral-500">Our services</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <ServiceCard title="Cattle tracking" description="Geofenced herds, RFID-tagged hooves, real-time strays." />
            <ServiceCard title="Saloon reservations" description="Book a private corner, pre-order a round." />
            <ServiceCard title="Bounty board" description="Post, bid, and settle bounties with escrow." />
          </div>
        </section>

        <section className="mt-12">
          <h2 className="text-sm font-medium uppercase tracking-wide text-neutral-500">Platform Mesh</h2>
          <div className="mt-4 rounded-lg border border-neutral-200 bg-white p-6 shadow-sm">
            <div className="text-base font-medium">Platform Mesh integration</div>
            <p className="mt-2 max-w-2xl text-sm text-neutral-600">
              Mint access tokens that let your Platform Mesh accounts consume
              Wildest Co. APIs directly. Manage tokens, see active bindings,
              and revoke anytime.
            </p>
            <button
              onClick={onOpenAdmin}
              className="mt-4 rounded bg-neutral-900 px-4 py-2 text-sm font-medium text-white hover:bg-neutral-800"
            >
              Manage PM integration
            </button>
          </div>
        </section>
      </main>
    </div>
  );
}

function ServiceCard({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-lg border border-neutral-200 bg-neutral-100/60 p-5 opacity-70">
      <div className="flex items-center justify-between">
        <div className="text-sm font-medium text-neutral-700">{title}</div>
        <span className="rounded bg-neutral-300 px-2 py-0.5 text-[10px] uppercase tracking-wide text-neutral-700">
          Soon
        </span>
      </div>
      <p className="mt-2 text-xs text-neutral-500">{description}</p>
    </div>
  );
}
