import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (username === 'admin' && password === 'admin') {
      localStorage.setItem('token', 'mock-jwt-token');
      navigate('/');
    } else {
      setError('用户名或密码错误，请重试');
    }
  };

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-primary)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: 16, padding: '40px 36px', width: 380 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{ fontSize: 36, marginBottom: 12 }}>📹</div>
          <h2 style={{ margin: 0, color: 'var(--text-primary)', fontSize: 20 }}>CaptureEngine</h2>
          <p style={{ margin: '6px 0 0', color: 'var(--text-muted)', fontSize: 13 }}>视频流抓拍引擎控制台</p>
        </div>
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <label style={{ fontSize: 11, color: 'var(--text-secondary)', display: 'block', marginBottom: 6, textTransform: 'uppercase' }}>用户名</label>
            <input placeholder="输入用户名" value={username} onChange={e => setUsername(e.target.value)} style={{ width: '100%' }} />
          </div>
          <div>
            <label style={{ fontSize: 11, color: 'var(--text-secondary)', display: 'block', marginBottom: 6, textTransform: 'uppercase' }}>密码</label>
            <input type="password" placeholder="输入密码" value={password} onChange={e => setPassword(e.target.value)} style={{ width: '100%' }} />
          </div>
          {error && (
            <div style={{ background: 'rgba(243,139,168,0.1)', border: '1px solid rgba(243,139,168,0.3)', borderRadius: 8, padding: '10px 14px', color: 'var(--red)', fontSize: 12 }}>
              ⚠️ {error}
            </div>
          )}
          <button type="submit" style={{ width: '100%', padding: 10, fontSize: 13, marginTop: 4 }}>登 录</button>
        </form>
        <div style={{ textAlign: 'center', marginTop: 24, fontSize: 11, color: 'var(--text-dim)' }}>
          v1.0.0 · 仅限授权用户访问
        </div>
      </div>
    </div>
  );
}
