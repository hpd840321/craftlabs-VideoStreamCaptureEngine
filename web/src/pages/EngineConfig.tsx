import { useAPI } from '../hooks/useAPI';

export default function EngineConfig() {
  const { data, loading } = useAPI<Record<string, unknown>>('/config');

  if (loading) return <div style={{ padding: 20, color: 'var(--text-muted)' }}>加载中...</div>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <h3 style={{ fontSize: 15, margin: 0 }}>引擎配置</h3>
        <button>💾 保存配置</button>
      </div>
      <div className="card">
        <div style={{ fontWeight: 600, marginBottom: 10, fontSize: 13 }}>引擎参数</div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          {['max_concurrent_starts','shutdown_timeout','health_check_interval','health_check_timeout'].map(k => (
            <div key={k}>
              <label style={{ fontSize: 10, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>{k}</label>
              <input defaultValue={String((data?.engine as Record<string,unknown>)?.[k] || '')} style={{ width: '100%' }} />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
