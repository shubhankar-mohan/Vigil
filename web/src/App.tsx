import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import DashboardPage from './pages/Dashboard';
import SwitchForm from './pages/SwitchForm';
import SwitchDetail from './pages/SwitchDetail';
import AutoRules from './pages/AutoRules';
import './App.css';

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <nav className="nav">
          <Link to="/" className="nav-brand">Vigil</Link>
          <div className="nav-links">
            <Link to="/">Dashboard</Link>
            <Link to="/switches/new">+ New Switch</Link>
            <Link to="/auto">Auto-Discovery</Link>
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
