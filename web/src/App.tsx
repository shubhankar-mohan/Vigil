import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom';
import DashboardPage from './pages/Dashboard';
import SwitchForm from './pages/SwitchForm';
import SwitchDetail from './pages/SwitchDetail';
import AutoRules from './pages/AutoRules';
import './App.css';

function NavLink({ to, children }: { to: string; children: React.ReactNode }) {
  const location = useLocation();
  const isActive = to === '/' ? location.pathname === '/' : location.pathname.startsWith(to);
  return <Link to={to} className={isActive ? 'active' : ''}>{children}</Link>;
}

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <nav className="nav">
          <Link to="/" className="nav-brand">Vigil</Link>
          <div className="nav-links">
            <NavLink to="/">Dashboard</NavLink>
            <NavLink to="/switches/new">+ New Switch</NavLink>
            <NavLink to="/auto">Auto-Discovery</NavLink>
          </div>
        </nav>
        <main className="main">
          <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/switches/new" element={<SwitchForm />} />
            <Route path="/switches/:id/edit" element={<SwitchForm />} />
            <Route path="/switches/:id" element={<SwitchDetail />} />
            <Route path="/auto" element={<AutoRules />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
