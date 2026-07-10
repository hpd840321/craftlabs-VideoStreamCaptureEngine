const mockStreams = [
  { id: 'gate-north', group: '园区-北门', status: 'running', fps: 25.0, res: '1920×1080', frames: '2.1M', latency: '42ms', uptime: '3d 14h' },
  { id: 'gate-south', group: '园区-南门', status: 'running', fps: 25.0, res: '1920×1080', frames: '1.9M', latency: '45ms', uptime: '3d 14h' },
  { id: 'lobby-main', group: '园区-北门', status: 'warning', fps: 18.7, res: '1920×1080', frames: '1.8M', latency: '847ms', uptime: '7d 2h' },
  { id: 'warehouse-03', group: '仓库', status: 'error', fps: 0, res: '1280×720', frames: '0', latency: '-', uptime: '重连中' },
  { id: 'rooftop-west', group: '停车场', status: 'stopped', fps: 0, res: '1920×1080', frames: '0', latency: '-', uptime: '-' },
];

const statusMap: Record<string, { color: string; label: string; dot: string }> = {
  running: { color: 'var(--green)', label: '运行中', dot: 'var(--green)' },
  warning: { color: 'var(--yellow)', label: '丢帧中', dot: 'var(--yellow)' },
  error: { color: 'var(--red)', label: '断流', dot: 'var(--red)' },
  stopped: { color: 'var(--text-muted)', label: '已停止', dot: 'var(--text-muted)' },
};

export default function StreamList() {
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
        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>匹配 <b style={{color:'var(--blue)'}}>5</b> · 共 50</span>
        <span style={{ color: 'var(--border)' }}>|</span>
        {['全部','运行中 42','告警 3','已停止 5'].map((t,i) => (
          <span key={t} style={{ padding: '3px 10px', borderRadius: 4, background: i===0?'rgba(137,180,250,0.12)':'transparent', border: i===0?'none':'1px solid var(--border)', color: i===0?'var(--blue)':'var(--text-secondary)', fontSize: 11, cursor: 'pointer', fontWeight: i===0?600:400 }}>{t}</span>
        ))}
        <span style={{ color: 'var(--border)' }}>|</span>
        <select style={{ fontSize: 11 }}><option>全部分组</option><option>园区-北门</option><option>仓库</option></select>
        <select style={{ fontSize: 11 }}><option>全部分辨率</option><option>1920×1080</option><option>1280×720</option></select>
      </div>

      <div style={{ border: '1px solid var(--border)', borderRadius: 8, overflow: 'hidden' }}>
        <table>
          <thead>
            <tr style={{ background: 'var(--bg-primary)' }}>
              <th>☐</th><th>流 ID</th><th>分组</th><th>状态</th><th>FPS</th><th>分辨率</th><th>出帧</th><th>延迟</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {mockStreams.map(s => {
              const st = statusMap[s.status];
              return (
                <tr key={s.id} style={{ background: s.status==='error'?'rgba(243,139,168,0.04)':s.status==='warning'?'rgba(249,226,175,0.04)':'transparent' }}>
                  <td><input type="checkbox" /></td>
                  <td style={{ fontWeight: 500 }}>{s.id}</td>
                  <td><span style={{ fontSize: 10, background: 'var(--border)', padding: '2px 6px', borderRadius: 3 }}>{s.group}</span></td>
                  <td><span style={{ width: 6, height: 6, background: st.dot, borderRadius: '50%', display: 'inline-block', marginRight: 5 }} /><span style={{ color: st.color, fontSize: 11 }}>{st.label}</span></td>
                  <td style={{ color: s.status==='warning'?'var(--yellow)':s.status==='running'?'var(--text-primary)':'var(--text-muted)' }}>{s.fps || '-'}</td>
                  <td style={{ color: 'var(--text-secondary)' }}>{s.res}</td>
                  <td>{s.frames}</td>
                  <td style={{ color: s.status==='running'?'var(--green)':'var(--text-muted)' }}>{s.latency}</td>
                  <td>
                    <a href={`/streams/${s.id}`} style={{ margin: '0 4px' }}>详情</a>
                    {s.status === 'stopped' || s.status === 'error' ? (
                      <a href="#" style={{ margin: '0 4px', color: 'var(--green)' }}>启动</a>
                    ) : (
                      <><a href="#" style={{ margin: '0 4px', color: 'var(--yellow)' }}>重启</a><a href="#" style={{ margin: '0 4px', color: 'var(--red)' }}>停止</a></>
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
