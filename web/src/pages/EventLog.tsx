import { useState } from 'react';
import { useAPI } from '../hooks/useAPI';
import { ackEvents, type EventItem } from '../api/events';

const levelStyles: Record<string, { bg: string; color: string; label: string }> = {
  error: { bg: 'rgba(243,139,168,0.15)', color: 'var(--red)', label: '错误' },
  warning: { bg: 'rgba(249,226,175,0.15)', color: 'var(--yellow)', label: '警告' },
  info: { bg: 'rgba(137,180,250,0.12)', color: 'var(--blue)', label: '信息' },
};

export default function EventLog() {
  const [level, setLevel] = useState('');
  const { data, loading, refetch } = useAPI<{ total: number; items: EventItem[] }>(
    `/events${level ? `?level=${level}` : ''}`,
    [level]
  );

  const handleAckAll = async () => {
    await ackEvents([]);
    refetch();
  };

  if (loading) return <div style={{ padding: 20, color: 'var(--text-muted)' }}>加载中...</div>;
  const items = data?.items || [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <div>
          <h3 style={{ fontSize: 15, margin: 0 }}>事件日志</h3>
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>共 {data?.total || 0} 条</span>
        </div>
        <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }} onClick={handleAckAll}>✅ 全部确认</button>
      </div>

      <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px' }}>
        <span style={{ fontSize: 10, color: 'var(--text-dim)', textTransform: 'uppercase' }}>级别</span>
        {[
          { label: '全部', value: '' },
          { label: '错误', value: 'error' },
          { label: '警告', value: 'warning' },
          { label: '信息', value: 'info' },
        ].map(t => (
          <span key={t.value} onClick={() => setLevel(t.value)}
            style={{ padding: '3px 10px', borderRadius: 4, cursor: 'pointer', fontSize: 11,
              background: level === t.value ? 'rgba(137,180,250,0.12)' : 'transparent',
              border: level === t.value ? 'none' : '1px solid var(--border)',
              color: level === t.value ? 'var(--blue)' : 'var(--text-secondary)',
              fontWeight: level === t.value ? 600 : 400 }}>
            {t.label}
          </span>
        ))}
      </div>

      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead><tr style={{ background: 'var(--bg-primary)' }}><th style={{ width: 100 }}>时间</th><th style={{ width: 60 }}>级别</th><th style={{ width: 110 }}>流</th><th>事件</th><th style={{ width: 60, textAlign: 'center' }}>状态</th></tr></thead>
          <tbody>
            {items.map(e => {
              const st = levelStyles[e.level] || levelStyles.info;
              return (
                <tr key={e.id} style={{ background: e.level === 'error' ? 'rgba(243,139,168,0.04)' : e.level === 'warning' ? 'rgba(249,226,175,0.04)' : 'transparent' }}>
                  <td style={{ color: 'var(--text-muted)', fontSize: 11 }}>{new Date(e.created_at).toLocaleTimeString()}</td>
                  <td><span style={{ background: st.bg, padding: '2px 6px', borderRadius: 3, color: st.color, fontWeight: 600, fontSize: 10 }}>{st.label}</span></td>
                  <td>{e.stream_id || '—'}</td>
                  <td>{e.message}</td>
                  <td style={{ textAlign: 'center' }}>
                    <span style={{ fontSize: 10, padding: '2px 7px', borderRadius: 3, background: e.acknowledged ? 'var(--border)' : 'rgba(243,139,168,0.15)', color: e.acknowledged ? 'var(--text-dim)' : 'var(--red)' }}>
                      {e.acknowledged ? '已确认' : '未确认'}
                    </span>
                  </td>
                </tr>
              );
            })}
            {items.length === 0 && <tr><td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: 30 }}>暂无事件</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
