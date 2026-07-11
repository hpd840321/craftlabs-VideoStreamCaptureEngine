import { request } from './client';

export interface EventItem {
  id: number;
  stream_id: string;
  level: string;
  message: string;
  acknowledged: boolean;
  created_at: string;
}

export interface EventListResponse {
  total: number;
  items: EventItem[];
}

export function listEvents(params?: Record<string, string>) {
  const qs = params ? '?' + new URLSearchParams(params).toString() : '';
  return request<EventListResponse>(`/events${qs}`);
}

export function ackEvents(ids: number[]) {
  return request<void>('/events/ack', {
    method: 'POST',
    body: JSON.stringify({ ids }),
  });
}
