import { ReactNode } from 'react';

export default function Header({ right }: { right?: ReactNode }) {
  return (
    <header className="border-b border-neutral-200 bg-white">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
        <div className="text-xl font-semibold tracking-tight">
          <span className="mr-2">🌵</span>
          Wildest Co.
          <span className="ml-2">🌵</span>
        </div>
        <div className="text-sm text-neutral-600">{right}</div>
      </div>
    </header>
  );
}
