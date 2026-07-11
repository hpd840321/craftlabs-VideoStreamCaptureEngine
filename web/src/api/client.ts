const BASE = '/api';

export async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token');
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (res.status === 401) {
    localStorage.removeItem('token');
    window.location.href = '/login';
  }
  const json = await res.json();
  if (json.code !== 0) throw new Error(json.message);
  return json.data as T;
}
