export default function EngineConfig() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <h3 style={{ fontSize: 15, margin: 0 }}>引擎配置</h3>
        <div style={{ display: 'flex', gap: 8 }}>
          <button style={{ background: 'var(--border)', color: 'var(--text-secondary)' }}>🔄 恢复默认</button>
          <button>💾 保存配置</button>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 14 }}>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div className="card">
            <div style={{ fontWeight: 600, marginBottom: 10, fontSize: 13 }}>引擎参数</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              {[['最大并发启动数','5'],['关闭超时(s)','30'],['健康检查间隔(s)','10'],['健康检查超时(s)','30']].map(([l,v]) => (
                <div key={l}>
                  <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>{l}</label>
                  <input defaultValue={v} style={{ width: '100%' }} />
                </div>
              ))}
            </div>
          </div>

          <div className="card">
            <div style={{ fontWeight: 600, marginBottom: 10, fontSize: 13 }}>Kafka 输出</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <div style={{ gridColumn: '1 / -1' }}>
                <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Broker 地址</label>
                <input defaultValue="10.0.1.10:9092, 10.0.1.11:9092" style={{ width: '100%' }} />
              </div>
              {[['Topic 前缀','video-frames'],['最大消息大小','1048576']].map(([l,v]) => (
                <div key={l}>
                  <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>{l}</label>
                  <input defaultValue={v} style={{ width: '100%' }} />
                </div>
              ))}
              <div>
                <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>ACK 级别</label>
                <select style={{ width: '100%' }}><option>1 — Leader 确认</option></select>
              </div>
              <div>
                <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>压缩</label>
                <select style={{ width: '100%' }}><option>snappy</option><option>none</option><option>gzip</option></select>
              </div>
            </div>
          </div>

          <div className="card">
            <div style={{ fontWeight: 600, marginBottom: 10, fontSize: 13 }}>序列化</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              {[['输出格式','jpeg'],['JPEG 质量','85']].map(([l,v]) => (
                <div key={l}>
                  <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>{l}</label>
                  {l==='输出格式' ? <select style={{ width:'100%' }}><option>jpeg</option><option>png</option></select> : <input defaultValue={v} style={{ width:'100%' }} />}
                </div>
              ))}
            </div>
          </div>

          <div className="card">
            <div style={{ fontWeight: 600, marginBottom: 10, fontSize: 13 }}>数据库</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
              {[['主机','your-postgres-host'],['端口','5432'],['用户名','admin'],['密码','••••••••••'],['数据库名','capture_engine'],['SSL','disable']].map(([l,v]) => (
                <div key={l}>
                  <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>{l}</label>
                  <input defaultValue={v} type={l==='密码'?'password':'text'} style={{ width: '100%' }} />
                </div>
              ))}
            </div>
          </div>
        </div>

        <div style={{ width: 340, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div className="card">
            <div style={{ fontWeight: 600, marginBottom: 10, fontSize: 13 }}>默认重启策略</div>
            {[['最大重试','20'],['初始退避(s)','1'],['最大退避(s)','60'],['倍增因子','2.0']].map(([l,v]) => (
              <div key={l} style={{ marginBottom: 8 }}>
                <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>{l}</label>
                <input defaultValue={v} style={{ width: '100%' }} />
              </div>
            ))}
            <div style={{ display: 'flex', alignItems: 'flex-end', gap: 3, height: 40, marginTop: 10 }}>
              {[8,14,22,38,70,100].map((h,i) => (
                <div key={i} style={{ flex: 1, height: `${h}%`, background: i>3?'var(--red)':i>2?'var(--yellow)':'var(--blue)', borderRadius: '1px 1px 0 0' }} />
              ))}
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 9, color: 'var(--text-dim)', marginTop: 4 }}>
              <span>1s</span><span>2s</span><span>4s</span><span>8s</span><span>16s</span><span>60s</span>
            </div>
          </div>

          <div className="card" style={{ flex: 1 }}>
            <div style={{ fontWeight: 600, marginBottom: 8, fontSize: 13 }}>YAML 预览</div>
            <pre style={{ fontSize: 10, color: 'var(--text-secondary)', background: 'var(--bg-primary)', padding: 10, borderRadius: 6, overflow: 'auto', lineHeight: 1.5 }}>{`engine:
  max_concurrent_starts: 5
  shutdown_timeout: 30s

output:
  backend: kafka
  kafka:
    brokers:
      - 10.0.1.10:9092
    topic_prefix: video-frames
  serializer:
    format: jpeg
    quality: 85

database:
  host: your-postgres-host
  port: 5432
  user: admin
  dbname: capture_engine`}</pre>
          </div>
        </div>
      </div>
    </div>
  );
}
