import { BarChart, Bar, XAxis, CartesianGrid, ResponsiveContainer } from 'recharts';

const mockData = [
  { time: '14:25', fps: 24.5 },
  { time: '14:26', fps: 25.0 },
  { time: '14:27', fps: 24.8 },
  { time: '14:28', fps: 25.0 },
  { time: '14:29', fps: 23.2 },
  { time: '14:30', fps: 25.0 },
  { time: '14:31', fps: 0 },
  { time: '14:32', fps: 24.5 },
];

export default function Dashboard() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', gap: 14 }}>
        {[
          { label: '在线 / 总数', value: '42 / 50', sub: '8 路离线', color: 'var(--green)' },
          { label: '今日出帧', value: '847K', sub: '↑ 12% vs 昨日', color: 'var(--blue)' },
          { label: '平均 FPS', value: '23.5', sub: '目标 25', color: 'var(--yellow)' },
          { label: '活跃告警', value: '3', sub: '2 未确认', color: 'var(--red)' },
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
          <div style={{ display: 'flex', gap: 4, fontSize: 11 }}>
            {['5m','15m','1h'].map(t => (
              <span key={t} style={{ padding: '3px 10px', borderRadius: 4, background: t==='1h'?'rgba(137,180,250,0.15)':'var(--border)', color: t==='1h'?'var(--blue)':'var(--text-secondary)' }}>{t}</span>
            ))}
          </div>
        </div>
        <ResponsiveContainer width="100%" height="85%">
          <BarChart data={mockData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-dim)' }} />
            <Bar dataKey="fps" fill="var(--blue)" radius={[2,2,0,0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 10 }}>
          <span style={{ fontWeight: 600, fontSize: 13 }}>最近事件</span>
          <span style={{ fontSize: 11, background: 'rgba(243,139,168,0.12)', padding: '3px 8px', borderRadius: 4, color: 'var(--red)' }}>未确认 2</span>
        </div>
        {[
          { time: '14:32:05', level: '断流', color: 'var(--red)', msg: 'gate-north — RTSP 连接超时，正在重连 (3/20)', tag: '未确认' },
          { time: '14:15:22', level: '丢帧', color: 'var(--yellow)', msg: 'lobby-main — 背压丢弃 247 帧', tag: '未确认' },
          { time: '13:48:11', level: '恢复', color: 'var(--text-muted)', msg: 'parking-east — 重连成功，解码恢复正常', tag: '已确认' },
        ].map((e,i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 10px', borderLeft: `3px solid ${e.color}`, marginBottom: 6, background: i<2?'rgba(243,139,168,0.04)':'transparent', borderRadius: 4 }}>
            <span style={{ fontSize: 11, color: 'var(--text-muted)', minWidth: 70 }}>{e.time}</span>
            <span style={{ fontSize: 11, color: e.color, fontWeight: 600, minWidth: 40 }}>{e.level}</span>
            <span style={{ fontSize: 11, flex: 1 }}>{e.msg}</span>
            <span style={{ fontSize: 10, padding: '2px 7px', borderRadius: 3, background: e.tag==='未确认'?'rgba(243,139,168,0.2)':'var(--border)', color: e.tag==='未确认'?'var(--red)':'var(--text-dim)' }}>{e.tag}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
