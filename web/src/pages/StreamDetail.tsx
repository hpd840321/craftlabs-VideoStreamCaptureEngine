import { useParams, Link } from 'react-router-dom';
import { LineChart, Line, XAxis, CartesianGrid, ResponsiveContainer } from 'recharts';

const chartData = [
  { time: '13:32', fps: 25 }, { time: '13:47', fps: 24.8 }, { time: '14:02', fps: 25 }, { time: '14:17', fps: 24.5 }, { time: '14:32', fps: 25 },
];

export default function StreamDetail() {
  const { id } = useParams();

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}>
          <Link to="/streams" style={{ color: 'var(--text-muted)' }}>← 流列表</Link>
          <span style={{ color: 'var(--border-light)' }}>/</span>
          <span style={{ fontWeight: 600 }}>{id}</span>
          <span style={{ width: 7, height: 7, background: 'var(--green)', borderRadius: '50%' }} />
          <span style={{ color: 'var(--green)', fontSize: 11 }}>运行中</span>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }}>🔄 重启</button>
          <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }}>⏹ 停止</button>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 14 }}>
        <div style={{ width: 340, flexShrink: 0 }}>
          <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
            <div style={{ padding: '8px 12px', borderBottom: '1px solid var(--border)', fontSize: 11, fontWeight: 600 }}>实时帧预览</div>
            <div style={{ height: 220, background: 'linear-gradient(135deg, #1a1a2e, #16213e, #0f3460)', position: 'relative' }}>
              <div style={{ position: 'absolute', bottom: 0, left: 0, right: 0, background: 'rgba(0,0,0,0.7)', padding: '4px 10px', display: 'flex', justifyContent: 'space-between', fontSize: 10, color: 'var(--text-secondary)' }}>
                <span>2026-07-10 14:32:05.247</span>
                <span>#2,147,833</span>
              </div>
            </div>
          </div>
        </div>

        <div className="card" style={{ flex: 1, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
          <div style={{ background: 'var(--bg-tertiary)', borderRadius: 6, padding: '8px 10px' }}>
            <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>FPS</div>
            <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--green)' }}>25.0</div>
          </div>
          <div style={{ background: 'var(--bg-tertiary)', borderRadius: 6, padding: '8px 10px' }}>
            <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>延迟</div>
            <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--blue)' }}>42ms</div>
          </div>
          <div style={{ background: 'var(--bg-tertiary)', borderRadius: 6, padding: '8px 10px' }}>
            <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>累计出帧</div>
            <div style={{ fontSize: 20, fontWeight: 700 }}>2,147,833</div>
          </div>
          <div style={{ background: 'var(--bg-tertiary)', borderRadius: 6, padding: '8px 10px' }}>
            <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>丢帧率</div>
            <div style={{ fontSize: 20, fontWeight: 700, color: 'var(--green)' }}>0.02%</div>
          </div>
        </div>

        <div className="card" style={{ width: 300, flexShrink: 0, fontSize: 10 }}>
          <div style={{ fontWeight: 600, marginBottom: 8, fontSize: 11 }}>配置摘要</div>
          {[
            ['RTSP','rtsp://10.0.2.101/...'],['分辨率','1920×1080'],['采集帧率','25 fps'],['输出 Topic','gate-north'],['分组','园区-北门']
          ].map(([k,v]) => (
            <div key={k} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <span style={{ color: 'var(--text-muted)' }}>{k}</span><span>{v}</span>
            </div>
          ))}
        </div>
      </div>

      <div style={{ display: 'flex', gap: 14 }}>
        <div className="card" style={{ flex: 1, height: 180 }}>
          <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8 }}>FPS 趋势</div>
          <ResponsiveContainer width="100%" height="85%">
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--text-dim)' }} />
              <Line type="monotone" dataKey="fps" stroke="var(--blue)" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className="card" style={{ width: 340, flexShrink: 0 }}>
          <div style={{ fontWeight: 600, fontSize: 13, marginBottom: 8 }}>过滤链</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11 }}>
            <div style={{ flex: 1, textAlign: 'center', padding: 6, background: 'var(--bg-tertiary)', borderRadius: 6 }}>
              <div style={{ color: 'var(--blue)', fontWeight: 600 }}>输入</div>
              <div style={{ fontWeight: 700 }}>25.0 fps</div>
            </div>
            <span>→</span>
            <div style={{ flex: 1, textAlign: 'center', padding: 6, background: 'var(--bg-tertiary)', borderRadius: 6 }}>
              <div style={{ color: 'var(--yellow)', fontWeight: 600 }}>去重</div>
              <div style={{ color: 'var(--yellow)', fontWeight: 700 }}>-2.1%</div>
            </div>
            <span>→</span>
            <div style={{ flex: 1, textAlign: 'center', padding: 6, background: 'var(--bg-tertiary)', borderRadius: 6 }}>
              <div style={{ fontWeight: 600 }}>Kafka</div>
              <div style={{ color: 'var(--green)', fontWeight: 700 }}>24.5 fps</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
