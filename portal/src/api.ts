export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

export async function api<T = unknown>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const r = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers ?? {}),
    },
  });
  if (r.status === 204) return undefined as T;
  const text = await r.text();
  const body = text ? JSON.parse(text) : null;
  if (!r.ok) {
    throw new ApiError(body?.error ?? r.statusText, r.status);
  }
  return body as T;
}

export interface Me {
  id: string;
  email: string;
  createdAt: string;
}
