import { request } from './client';

export function getConfig() {
  return request<Record<string, unknown>>('/config');
}

export function updateConfig(config: Record<string, unknown>) {
  return request<Record<string, unknown>>('/config', {
    method: 'PUT',
    body: JSON.stringify(config),
  });
}
