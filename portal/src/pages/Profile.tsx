import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api';

interface Contact {
  fullName: string;
  company: string;
  supportEmail: string;
  updatedAt?: string;
}

interface Payment {
  cardHolder: string;
  cardLast4: string;
  expMonth: number;
  expYear: number;
  updatedAt?: string;
}

export default function Profile() {
  return (
    <section className="max-w-xl">
      <h2 className="text-lg font-semibold">Profile</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Placeholder pages — the portal demo doesn't gate anything on these.
      </p>
      <div className="mt-6 space-y-10">
        <ContactSection />
        <PaymentSection />
      </div>
    </section>
  );
}

function ContactSection() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ['contact'], queryFn: () => api<Contact>('/api/profile/contact') });
  const [form, setForm] = useState<Contact>({ fullName: '', company: '', supportEmail: '' });
  const [status, setStatus] = useState<string | null>(null);

  useEffect(() => {
    if (q.data) setForm(q.data);
  }, [q.data]);

  const save = useMutation({
    mutationFn: (v: Contact) =>
      api<Contact>('/api/profile/contact', { method: 'PUT', body: JSON.stringify(v) }),
    onSuccess: (data) => {
      qc.setQueryData(['contact'], data);
      setStatus('Saved.');
      setTimeout(() => setStatus(null), 2000);
    },
    onError: (e: Error) => setStatus(`Error: ${e.message}`),
  });

  return (
    <div>
      <h3 className="text-sm font-medium uppercase tracking-wide text-neutral-500">Contact</h3>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate(form);
        }}
        className="mt-3 space-y-4"
      >
        <Field label="Full name" value={form.fullName} onChange={(v) => setForm({ ...form, fullName: v })} />
        <Field label="Company" value={form.company} onChange={(v) => setForm({ ...form, company: v })} />
        <Field
          label="Support email"
          type="email"
          value={form.supportEmail}
          onChange={(v) => setForm({ ...form, supportEmail: v })}
        />
        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={save.isPending}
            className="rounded bg-neutral-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
          >
            {save.isPending ? 'Saving…' : 'Save contact'}
          </button>
          {status && <span className="text-sm text-neutral-600">{status}</span>}
        </div>
      </form>
    </div>
  );
}

function PaymentSection() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ['payment'], queryFn: () => api<Payment>('/api/profile/payment') });
  const [holder, setHolder] = useState('');
  const [number, setNumber] = useState('');
  const [expMonth, setExpMonth] = useState('');
  const [expYear, setExpYear] = useState('');
  const [status, setStatus] = useState<string | null>(null);

  const onFile = q.data && q.data.cardLast4;

  const save = useMutation({
    mutationFn: () =>
      api<Payment>('/api/profile/payment', {
        method: 'PUT',
        body: JSON.stringify({
          cardHolder: holder,
          cardNumber: number,
          expMonth: Number(expMonth),
          expYear: Number(expYear),
        }),
      }),
    onSuccess: (data) => {
      qc.setQueryData(['payment'], data);
      setHolder('');
      setNumber('');
      setExpMonth('');
      setExpYear('');
      setStatus('Saved.');
      setTimeout(() => setStatus(null), 2000);
    },
    onError: (e: Error) => setStatus(`Error: ${e.message}`),
  });

  return (
    <div>
      <h3 className="text-sm font-medium uppercase tracking-wide text-neutral-500">Payment</h3>
      {onFile && (
        <div className="mt-3 rounded border border-emerald-200 bg-emerald-50 p-3 text-sm">
          <div className="font-medium text-emerald-900">Payment method on file</div>
          <div className="mt-1 text-emerald-800">
            {q.data!.cardHolder} • **** {q.data!.cardLast4} • {q.data!.expMonth.toString().padStart(2, '0')}/
            {q.data!.expYear}
          </div>
        </div>
      )}
      <div className="mt-3 text-xs text-neutral-500">
        Demo only — no card data is sent to a payment processor. Only the last 4
        digits and holder name are stored.
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate();
        }}
        className="mt-3 space-y-4"
      >
        <Field label="Cardholder name" value={holder} onChange={setHolder} />
        <Field label="Card number" value={number} onChange={setNumber} placeholder="4111 1111 1111 1111" />
        <div className="grid grid-cols-2 gap-3">
          <Field label="Exp month" value={expMonth} onChange={setExpMonth} placeholder="12" />
          <Field label="Exp year" value={expYear} onChange={setExpYear} placeholder="2030" />
        </div>
        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={save.isPending}
            className="rounded bg-neutral-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
          >
            {save.isPending ? 'Saving…' : onFile ? 'Replace card' : 'Save payment'}
          </button>
          {status && <span className="text-sm text-neutral-600">{status}</span>}
        </div>
      </form>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  type = 'text',
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-neutral-700">{label}</span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="mt-1 w-full rounded border border-neutral-300 px-3 py-2 text-sm focus:border-neutral-500 focus:outline-none"
      />
    </label>
  );
}
