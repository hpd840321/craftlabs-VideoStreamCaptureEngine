import { useParams, Link } from 'react-router-dom';
import { LineChart, Line, XAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import { useAPI } from '../hooks/useAPI';
import type { StreamInfo } from '../api/streams';

export default function StreamDetail() {
  const { id } = useParams();
  const { data, loading } = useAPI<StreamInfo>(`/streams/${id}`);

  if (loading) return <div style={{ padding: 20, color: 'var(--text-muted)' }}>加载中...</div>;
  if (!data) return <div style={{ padding: 20, color: 'var(--red)' }}>流未找到</div>;

  const chartData = [{ time: '13:32', fps: 25 }, { time: '13:47', fps: data.fps || 24.8 }, { time: '14:02', fps: 25 }, { time: '14:17', fps: 24.5 }, { time: '14:32', fps: 25 }];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}>
          <Link to="/streams" style={{ color: 'var(--text-muted)' }}>← 流列表</Link>
          <span style={{ color: 'var(--border-light)' }}>/</span>
          <span style={{ fontWeight: 600 }}>{data.id}</span>
          <span style={{ color: 'var(--green)', fontSize: 11 }}>{data.status}</span>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }}>🔄 重启</button>
          <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }}>⏹ 停止</button>
        </div>
      </div>
      <div style={{ display: 'flex', gap: 14 }}>
        <div style={{ width: 340, flexShrink: 0 }} className="card" >
          <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8 }}>配置摘要</div>
          {[['RTSP', data.rtsp_url || ''],['分辨率', data.resolution],['采集帧率', `${data.capture_fps} fps`],['输出 Topic', data.output_topic],['分组', data.group]].map(([k,v]) => (
            <div key={k} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4, fontSize: 11 }}>
              <span style={{ color: 'var(--text-muted)' }}>{k}</span><span>{v || '-'}</span>
            </div>
          ))}
        </div>
        <div className="card" style={{ flex: 1 }}>
          <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8 }}>FPS 趋势</div>
          <ResponsiveContainer width="100%" height={180}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-dim)' }} />
              <Line type="monotone" dataKey="fps" stroke="var(--blue)" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
