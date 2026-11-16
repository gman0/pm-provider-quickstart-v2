import { createContext, useContext, ReactNode } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiError, Me } from './api';

interface AuthContextValue {
  me: Me | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  deleteAccount: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient();

  const meQuery = useQuery<Me | null>({
    queryKey: ['me'],
    queryFn: async () => {
      try {
        return await api<Me>('/api/auth/me');
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    staleTime: 60_000,
  });

  const loginMut = useMutation({
    mutationFn: (v: { email: string; password: string }) =>
      api<Me>('/api/auth/login', { method: 'POST', body: JSON.stringify(v) }),
    onSuccess: (data) => qc.setQueryData(['me'], data),
  });

  const registerMut = useMutation({
    mutationFn: (v: { email: string; password: string }) =>
      api<Me>('/api/auth/register', { method: 'POST', body: JSON.stringify(v) }),
    onSuccess: (data) => qc.setQueryData(['me'], data),
  });

  const logoutMut = useMutation({
    mutationFn: () => api('/api/auth/logout', { method: 'POST' }),
    onSuccess: () => qc.setQueryData(['me'], null),
  });

  const deleteMut = useMutation({
    mutationFn: () => api('/api/auth/me', { method: 'DELETE' }),
    onSuccess: () => qc.setQueryData(['me'], null),
  });

  const value: AuthContextValue = {
    me: meQuery.data ?? null,
    loading: meQuery.isLoading,
    login: async (email, password) => {
      await loginMut.mutateAsync({ email, password });
    },
    register: async (email, password) => {
      await registerMut.mutateAsync({ email, password });
    },
    logout: async () => {
      await logoutMut.mutateAsync();
    },
    deleteAccount: async () => {
      await deleteMut.mutateAsync();
    },
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth outside AuthProvider');
  return ctx;
}
