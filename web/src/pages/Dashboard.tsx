import { BarChart, Bar, XAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import { useAPI } from '../hooks/useAPI';

interface MetricsSummary {
  online_streams: number; total_streams: number; frames_today: number;
  avg_fps: number; active_alerts: number; unacknowledged: number; fps_trend: number[];
}

export default function Dashboard() {
  const { data, loading } = useAPI<MetricsSummary>('/metrics/summary');
  if (loading) return <div style={{ padding: 20, color: 'var(--text-muted)' }}>加载中...</div>;
  if (!data) return <div style={{ padding: 20, color: 'var(--red)' }}>加载失败</div>;

  const chartData = (data.fps_trend || []).map((fps, i) => ({
    time: `${14 + Math.floor(i/6)}:${String((i%6)*10).padStart(2,'0')}`,
    fps,
  }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', gap: 14 }}>
        {[
          { label: '在线 / 总数', value: `${data.online_streams} / ${data.total_streams}`, sub: `${data.total_streams - data.online_streams} 路离线`, color: 'var(--green)' },
          { label: '今日出帧', value: `${(data.frames_today / 1000).toFixed(0)}K`, sub: '实时累计', color: 'var(--blue)' },
          { label: '平均 FPS', value: String(data.avg_fps), sub: '目标 25', color: 'var(--yellow)' },
          { label: '活跃告警', value: String(data.active_alerts), sub: `${data.unacknowledged} 未确认`, color: 'var(--red)' },
        ].map(s => (
          <div key={s.label} className="card" style={{ flex: 1 }}>
            <div style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase', marginBottom: 4 }}>{s.label}</div>
            <div style={{ fontSize: 28, fontWeight: 700, color: s.color }}>{s.value}</div>
            <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{s.sub}</div>
          </div>
        ))}
      </div>
      <div className="card" style={{ height: 220 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
          <span style={{ fontWeight: 600, fontSize: 13 }}>实时 FPS 趋势</span>
          <span style={{ fontSize: 11, padding: '3px 10px', borderRadius: 4, background: 'rgba(137,180,250,0.15)', color: 'var(--blue)' }}>1h</span>
        </div>
        <ResponsiveContainer width="100%" height="85%">
          <BarChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-dim)' }} />
            <Bar dataKey="fps" fill="var(--blue)" radius={[2,2,0,0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
