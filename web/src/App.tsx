import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import StreamList from './pages/StreamList';
import StreamDetail from './pages/StreamDetail';
import EngineConfig from './pages/EngineConfig';
import EventLog from './pages/EventLog';

function AuthGuard({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token');
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<AuthGuard><Layout /></AuthGuard>}>
          <Route index element={<Dashboard />} />
          <Route path="streams" element={<StreamList />} />
          <Route path="streams/:id" element={<StreamDetail />} />
          <Route path="config" element={<EngineConfig />} />
          <Route path="events" element={<EventLog />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
