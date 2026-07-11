import { request } from './client';

export interface StreamInfo {
  id: string;
  group: string;
  status: string;
  fps: number;
  resolution: string;
  frames_total: number;
  latency_ms: string;
  uptime: string;
  rtsp_url: string;
  output_topic: string;
  capture_fps: number;
  decode_scale: string;
}

export interface StreamListResponse {
  total: number;
  page: number;
  size: number;
  items: StreamInfo[];
}

export function listStreams(params?: Record<string, string>) {
  const qs = params ? '?' + new URLSearchParams(params).toString() : '';
  return request<StreamListResponse>(`/streams${qs}`);
}

export function getStream(id: string) {
  return request<StreamInfo>(`/streams/${id}`);
}

export function createStream(stream: Partial<StreamInfo>) {
  return request<StreamInfo>('/streams', {
    method: 'POST',
    body: JSON.stringify(stream),
  });
}

export function updateStream(id: string, stream: Partial<StreamInfo>) {
  return request<StreamInfo>(`/streams/${id}`, {
    method: 'PUT',
    body: JSON.stringify(stream),
  });
}

export function deleteStream(id: string) {
  return request<void>(`/streams/${id}`, { method: 'DELETE' });
}

export function streamAction(id: string, action: 'start' | 'stop' | 'restart') {
  return request<{ action: string; stream_id: string }>(`/streams/${id}/${action}`, { method: 'POST' });
}
