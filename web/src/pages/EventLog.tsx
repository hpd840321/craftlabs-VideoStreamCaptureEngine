const events = [
  { time: '14:32:05', level: 'error', stream: 'gate-north', msg: 'RTSP 连接超时，ffmpeg 进程退出 (exit=1)，开始重连', ack: false },
  { time: '14:15:22', level: 'warning', stream: 'lobby-main', msg: '背压丢帧 — Kafka 写入延迟 847ms，丢弃 247 帧', ack: false },
  { time: '13:48:11', level: 'info', stream: 'parking-east', msg: '重连成功，解码恢复正常 (25.0 fps)', ack: true },
  { time: '13:15:00', level: 'info', stream: '—', msg: '配置热更新 — gate-north DuplicateFilter threshold 10 → 8', ack: true },
  { time: '12:55:18', level: 'warning', stream: 'warehouse-03', msg: 'FPS 低于阈值 — 当前 8.3 (阈值 15)', ack: true },
];

const levelStyles: Record<string, { bg: string; color: string; label: string }> = {
  error: { bg: 'rgba(243,139,168,0.15)', color: 'var(--red)', label: '错误' },
  warning: { bg: 'rgba(249,226,175,0.15)', color: 'var(--yellow)', label: '警告' },
  info: { bg: 'rgba(137,180,250,0.12)', color: 'var(--blue)', label: '信息' },
};

export default function EventLog() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <div>
          <h3 style={{ fontSize: 15, margin: 0 }}>事件日志</h3>
          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>共 1,247 条 · 2 条未确认告警</span>
        </div>
        <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }}>✅ 全部确认</button>
      </div>

      <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', padding: '10px 14px' }}>
        <span style={{ fontSize: 10, color: 'var(--text-dim)', textTransform: 'uppercase' }}>级别</span>
        {['全部','错误 23','警告 89','信息 1,135'].map((t,i) => (
          <span key={t} style={{ padding: '3px 10px', borderRadius: 4, background: i===0?'rgba(137,180,250,0.12)':'transparent', border: i===0?'none':'1px solid var(--border)', color: i===0?'var(--blue)':i===2?'var(--yellow)':'var(--text-secondary)', fontSize: 11 }}>{t}</span>
        ))}
        <span style={{ color: 'var(--border)' }}>|</span>
        <select style={{ fontSize: 11 }}><option>全部流</option></select>
        <select style={{ fontSize: 11 }}><option>最近 1 小时</option></select>
        <input placeholder="🔍 搜索事件..." style={{ width: 180, marginLeft: 'auto' }} />
      </div>

      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead><tr style={{ background: 'var(--bg-primary)' }}><th style={{ width: 100 }}>时间</th><th style={{ width: 60 }}>级别</th><th style={{ width: 110 }}>流</th><th>事件</th><th style={{ width: 60, textAlign: 'center' }}>状态</th></tr></thead>
          <tbody>
            {events.map((e,i) => {
              const st = levelStyles[e.level];
              return (
                <tr key={i} style={{ background: e.level==='error'?'rgba(243,139,168,0.04)':e.level==='warning'?'rgba(249,226,175,0.04)':'transparent' }}>
                  <td style={{ color: 'var(--text-muted)', fontSize: 11 }}>{e.time}</td>
                  <td><span style={{ background: st.bg, padding: '2px 6px', borderRadius: 3, color: st.color, fontWeight: 600, fontSize: 10 }}>{st.label}</span></td>
                  <td style={{ color: e.stream==='—'?'var(--text-muted)':'var(--text-primary)' }}>{e.stream}</td>
                  <td>{e.msg}</td>
                  <td style={{ textAlign: 'center' }}>
                    <span style={{ fontSize: 10, padding: '2px 7px', borderRadius: 3, background: e.ack?'var(--border)':'rgba(243,139,168,0.15)', color: e.ack?'var(--text-dim)':'var(--red)' }}>
                      {e.ack ? '已确认' : '未确认'}
                    </span>
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
