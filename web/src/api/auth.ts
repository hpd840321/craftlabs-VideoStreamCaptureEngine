import { request } from './client';

interface LoginResponse {
  token: string;
  expires_at: string;
}

export function login(username: string, password: string) {
  return request<LoginResponse>('/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
}

export function refresh() {
  return request<LoginResponse>('/refresh', { method: 'POST' });
}
