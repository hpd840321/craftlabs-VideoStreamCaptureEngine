import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import styles from './Layout.module.css';

const NAV_ITEMS = [
  { to: '/', label: '📊 仪表盘' },
  { to: '/streams', label: '📹 流管理' },
  { to: '/config', label: '⚙️ 引擎配置' },
  { to: '/events', label: '📋 事件日志' },
];

export default function Layout() {
  const navigate = useNavigate();

  const handleLogout = () => {
    localStorage.removeItem('token');
    navigate('/login');
  };

  return (
    <div className={styles.wrapper}>
      <header className={styles.topbar}>
        <div className={styles.brand}>CaptureEngine</div>
        <div className={styles.status}>
          <span className={styles.dot} /> 系统正常 | 42/50 在线
        </div>
        <div className={styles.actions}>
          <button className={styles.iconBtn}>🔔</button>
          <button className={styles.iconBtn}>⚙️</button>
          <div className={styles.avatar} onClick={handleLogout} style={{cursor:'pointer'}} title="退出登录">Z</div>
        </div>
      </header>
      <div className={styles.body}>
        <nav className={styles.sidebar}>
          {NAV_ITEMS.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
              }
            >
              {item.label}
            </NavLink>
          ))}
          <div className={styles.version}>v1.0.0</div>
        </nav>
        <main className={styles.content}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
