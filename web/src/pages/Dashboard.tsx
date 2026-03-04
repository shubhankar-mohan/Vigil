import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { Dashboard } from '../api';

function timeAgo(dateStr: string | null): string {
  if (!dateStr) return '-';
  const diff = Date.now() - new Date(dateStr).getTime();
  if (diff < 0) return 'just now';
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function timeUntil(dateStr: string | null): string {
  if (!dateStr) return '-';
  const diff = new Date(dateStr).getTime() - Date.now();
  if (diff < 0) return timeAgo(dateStr);
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `in ${mins}m`;
  const hrs = Math.floor(mins / 60);
  return `in ${hrs}h ${mins % 60}m`;
}

function badgeClass(state: string): string {
  return `badge badge-${state}`;
}

export default function DashboardPage() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [error, setError] = useState('');

  const load = () => {
    api.dashboard().then(setData).catch(e => setError(e.message));
  };

  useEffect(() => {
    load();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, []);

  if (error) return <div className="error">{error}</div>;
  if (!data) return <p className="muted">Loading...</p>;

  return (
    <div>
      <div className="page-header">
        <h1>Switches</h1>
        <Link to="/switches/new" className="btn btn-primary">+ New Switch</Link>
      </div>

      <div className="stats">
        <div className="stat-card"><div className="number">{data.total}</div><div className="label">Total</div></div>
        <div className="stat-card"><div className="number" style={{color:'#40c057'}}>{data.up}</div><div className="label">Up</div></div>
        <div className="stat-card"><div className="number" style={{color:'#f03e3e'}}>{data.down}</div><div className="label">Down</div></div>
        <div className="stat-card"><div className="number" style={{color:'#fab005'}}>{data.grace}</div><div className="label">Grace</div></div>
        <div className="stat-card"><div className="number" style={{color:'#22b8cf'}}>{data.learning}</div><div className="label">Learning</div></div>
        <div className="stat-card"><div className="number" style={{color:'#868e96'}}>{data.paused}</div><div className="label">Paused</div></div>
      </div>

      {data.switches.length === 0 ? (
        <p className="muted">No switches configured. Create one to get started.</p>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Last Seen</th>
              <th>Expected</th>
              <th>Mode</th>
              <th>Signal</th>
            </tr>
          </thead>
          <tbody>
            {data.switches.map(sw => (
              <tr key={sw.id}>
                <td><Link to={`/switches/${sw.id}`}>{sw.name}</Link></td>
                <td><span className={badgeClass(sw.state)}>{sw.state}</span></td>
                <td>{timeAgo(sw.last_signal_at)}</td>
                <td>{timeUntil(sw.next_expected_at)}</td>
                <td>{sw.mode}</td>
                <td>{sw.signal}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
