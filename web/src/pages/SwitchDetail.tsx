import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api';
import type { Switch, EvalHistory } from '../api';

function formatTime(dateStr: string | null): string {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString();
}

function timeAgo(dateStr: string | null): string {
  if (!dateStr) return '-';
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ${mins % 60}m ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export default function SwitchDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [sw, setSw] = useState<Switch | null>(null);
  const [history, setHistory] = useState<EvalHistory[]>([]);
  const [error, setError] = useState('');

  const load = () => {
    const swId = Number(id);
    api.getSwitch(swId).then(setSw).catch(e => setError(e.message));
    api.getSwitchHistory(swId).then(setHistory).catch(() => {});
  };

  useEffect(() => {
    load();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, [id]);

  const handlePause = async () => {
    await api.pauseSwitch(Number(id));
    load();
  };

  const handleResume = async () => {
    await api.resumeSwitch(Number(id));
    load();
  };

  const handleDelete = async () => {
    if (!confirm('Delete this switch?')) return;
    await api.deleteSwitch(Number(id));
    navigate('/');
  };

  if (error) return <div className="error">{error}</div>;
  if (!sw) return <p className="muted">Loading...</p>;

  return (
    <div>
      <div className="detail-header">
        <h1>{sw.name}</h1>
        <span className={`badge badge-${sw.state}`}>{sw.state}</span>
        {sw.auto_created && <span className="badge badge-learning">auto</span>}
      </div>

      <div className="detail-grid">
        <div className="detail-item">
          <label>Last Signal</label>
          <div className="value">{formatTime(sw.last_signal_at)} ({timeAgo(sw.last_signal_at)})</div>
        </div>
        <div className="detail-item">
          <label>Next Expected</label>
          <div className="value">{formatTime(sw.next_expected_at)}</div>
        </div>
        <div className="detail-item">
          <label>In State Since</label>
          <div className="value">{formatTime(sw.state_changed_at)} ({timeAgo(sw.state_changed_at)})</div>
        </div>
        <div className="detail-item">
          <label>Mode / Signal</label>
          <div className="value">{sw.mode} / {sw.signal}</div>
        </div>
        <div className="detail-item">
          <label>Query</label>
          <div className="value" style={{fontSize: '0.8rem', wordBreak: 'break-all'}}>{sw.query}</div>
        </div>
        <div className="detail-item">
          <label>Evals</label>
          <div className="value" style={{color:'#40c057'}}>{sw.eval_pass_count} pass <span style={{color:'#f03e3e'}}>{sw.eval_fail_count} fail</span></div>
        </div>
        {sw.mode === 'frequency' && (
          <>
            <div className="detail-item">
              <label>Interval</label>
              <div className="value">{sw.interval_seconds}s ({Math.round(sw.interval_seconds / 60)}m)</div>
            </div>
            <div className="detail-item">
              <label>Grace</label>
              <div className="value">{sw.grace_seconds}s</div>
            </div>
          </>
        )}
        {sw.mode === 'irregularity' && (
          <>
            <div className="detail-item">
              <label>Min Samples</label>
              <div className="value">{sw.min_samples}</div>
            </div>
            <div className="detail-item">
              <label>Tolerance</label>
              <div className="value">{sw.tolerance_multiplier}x median</div>
            </div>
          </>
        )}
      </div>

      <div className="btn-group">
        <Link to={`/switches/${sw.id}/edit`} className="btn">Edit</Link>
        {sw.state === 'paused'
          ? <button className="btn" onClick={handleResume}>Resume</button>
          : <button className="btn" onClick={handlePause}>Pause</button>
        }
        <button className="btn btn-danger" onClick={handleDelete}>Delete</button>
      </div>

      <h2 style={{marginTop: '2rem', marginBottom: '0.5rem'}}>Evaluation History</h2>
      {history.length === 0 ? (
        <p className="muted">No evaluations yet.</p>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Result</th>
              <th>State</th>
              <th>Details</th>
            </tr>
          </thead>
          <tbody>
            {history.map(h => (
              <tr key={h.id}>
                <td>{new Date(h.eval_at).toLocaleString()}</td>
                <td><span className={`badge badge-${h.result === 'pass' ? 'up' : 'down'}`}>{h.result}</span></td>
                <td>{h.state}</td>
                <td style={{fontSize: '0.8rem', color: '#8b949e'}}>{h.details}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
