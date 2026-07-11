import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAPI } from '../hooks/useAPI';
import { streamAction, type StreamInfo } from '../api/streams';

const statusMap: Record<string, { color: string; label: string; dot: string }> = {
  running: { color: 'var(--green)', label: '运行中', dot: 'var(--green)' },
  warning: { color: 'var(--yellow)', label: '丢帧中', dot: 'var(--yellow)' },
  error: { color: 'var(--red)', label: '断流', dot: 'var(--red)' },
  stopped: { color: 'var(--text-muted)', label: '已停止', dot: 'var(--text-muted)' },
};

export default function StreamList() {
  const [status, setStatus] = useState('');
  const { data, loading, refetch } = useAPI<{ total: number; items: StreamInfo[] }>(
    `/streams${status ? `?status=${status}` : ''}`,
    [status]
  );

  const handleAction = async (id: string, action: 'start' | 'stop' | 'restart') => {
    await streamAction(id, action);
    refetch();
  };

  if (loading) return <div style={{ padding: 20, color: 'var(--text-muted)' }}>加载中...</div>;

  const items = data?.items || [];
  const total = data?.total || 0;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <h3 style={{ fontSize: 15, margin: 0 }}>流列表</h3>
        <div style={{ display: 'flex', gap: 8 }}>
          <button>+ 添加流</button>
          <button style={{ background: 'rgba(137,180,250,0.15)', color: 'var(--blue)' }}>📥 批量导入</button>
        </div>
      </div>

      <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', padding: '10px 14px' }}>
        <input placeholder="🔍 搜索流 ID、RTSP 地址..." style={{ width: 260 }} />
        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>共 {total}</span>
        <span style={{ color: 'var(--border)' }}>|</span>
        {[
          { label: '全部', value: '' },
          { label: '运行中', value: 'running' },
          { label: '告警', value: 'warning' },
          { label: '已停止', value: 'stopped' },
        ].map(t => (
          <span key={t.value} onClick={() => setStatus(t.value)}
            style={{ padding: '3px 10px', borderRadius: 4, cursor: 'pointer', fontSize: 11,
              background: status === t.value ? 'rgba(137,180,250,0.12)' : 'transparent',
              border: status === t.value ? 'none' : '1px solid var(--border)',
              color: status === t.value ? 'var(--blue)' : 'var(--text-secondary)',
              fontWeight: status === t.value ? 600 : 400 }}>
            {t.label}
          </span>
        ))}
      </div>

      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead><tr style={{ background: 'var(--bg-primary)' }}>
            <th>流 ID</th><th>分组</th><th>状态</th><th>FPS</th><th>分辨率</th><th>出帧</th><th>延迟</th><th>操作</th>
          </tr></thead>
          <tbody>
            {items.map(s => {
              const st = statusMap[s.status] || { color: 'var(--text-muted)', label: s.status, dot: 'var(--text-muted)' };
              return (
                <tr key={s.id} style={{ background: s.status === 'error' ? 'rgba(243,139,168,0.04)' : s.status === 'warning' ? 'rgba(249,226,175,0.04)' : 'transparent' }}>
                  <td style={{ fontWeight: 500 }}>{s.id}</td>
                  <td><span style={{ fontSize: 10, background: 'var(--border)', padding: '2px 6px', borderRadius: 3 }}>{s.group}</span></td>
                  <td><span style={{ width: 6, height: 6, background: st.dot, borderRadius: '50%', display: 'inline-block', marginRight: 5 }} /><span style={{ color: st.color, fontSize: 11 }}>{st.label}</span></td>
                  <td>{s.fps || '-'}</td>
                  <td style={{ color: 'var(--text-secondary)' }}>{s.resolution}</td>
                  <td>{s.frames_total?.toLocaleString() || '0'}</td>
                  <td style={{ color: s.status === 'running' ? 'var(--green)' : 'var(--text-muted)' }}>{s.latency_ms || '-'}</td>
                  <td>
                    <Link to={`/streams/${s.id}`} style={{ margin: '0 4px' }}>详情</Link>
                    {s.status === 'stopped' || s.status === 'error' ? (
                      <a href="#" onClick={e => { e.preventDefault(); handleAction(s.id, 'start'); }} style={{ margin: '0 4px', color: 'var(--green)' }}>启动</a>
                    ) : (
                      <><a href="#" onClick={e => { e.preventDefault(); handleAction(s.id, 'stop'); }} style={{ margin: '0 4px', color: 'var(--red)' }}>停止</a></>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
