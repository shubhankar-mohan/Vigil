import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api';
import type { Switch, EvalHistory } from '../api';

const RANGES = [
  { label: '30m', value: '30m' },
  { label: '1h', value: '1h' },
  { label: '3h', value: '3h' },
  { label: '6h', value: '6h' },
  { label: '12h', value: '12h' },
  { label: '1d', value: '24h' },
  { label: '1w', value: '168h' },
];

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

function resultColor(result: string): string {
  return result === 'pass' ? '#40c057' : '#f03e3e';
}

/* ── Uptime Bar ── */
function UptimeBar({ history, uptimePct }: { history: EvalHistory[]; uptimePct: number }) {
  if (history.length === 0) return null;

  // history is newest-first; reverse for left-to-right chronological
  const sorted = [...history].reverse();
  const segments = sorted.length;

  return (
    <div className="uptime-bar-container">
      <div className="uptime-bar-header">
        <span className="uptime-pct" style={{ color: uptimePct >= 99 ? '#40c057' : uptimePct >= 95 ? '#fab005' : '#f03e3e' }}>
          {uptimePct.toFixed(2)}%
        </span>
        <span className="uptime-label">uptime</span>
      </div>
      <div className="uptime-bar">
        {sorted.map((h) => (
          <div
            key={h.id}
            className="uptime-segment"
            style={{
              width: `${100 / segments}%`,
              backgroundColor: resultColor(h.result),
              opacity: 0.85,
            }}
            title={`${new Date(h.eval_at).toLocaleString()} — ${h.result} (${h.state})`}
          />
        ))}
      </div>
      <div className="uptime-bar-footer">
        <span className="muted">{formatTime(sorted[0]?.eval_at)}</span>
        <span className="muted">{formatTime(sorted[sorted.length - 1]?.eval_at)}</span>
      </div>
    </div>
  );
}

/* ── SVG Status Chart ── */
function StatusChart({ history }: { history: EvalHistory[] }) {
  if (history.length < 2) return <p className="muted">Not enough data for chart.</p>;

  const sorted = [...history].reverse();
  const W = 900, H = 200, PAD_L = 50, PAD_R = 20, PAD_T = 20, PAD_B = 40;
  const plotW = W - PAD_L - PAD_R;
  const plotH = H - PAD_T - PAD_B;

  const times = sorted.map(h => new Date(h.eval_at).getTime());
  const tMin = times[0], tMax = times[times.length - 1];
  const tRange = tMax - tMin || 1;

  const x = (t: number) => PAD_L + ((t - tMin) / tRange) * plotW;
  const y = (result: string) => result === 'pass' ? PAD_T + 20 : PAD_T + plotH - 20;

  // Build area segments colored by result
  const segments: { points: string; color: string }[] = [];
  for (let i = 0; i < sorted.length - 1; i++) {
    const x1 = x(times[i]), x2 = x(times[i + 1]);
    const y1 = y(sorted[i].result);
    const yBottom = PAD_T + plotH;
    const color = resultColor(sorted[i].result);
    segments.push({
      points: `${x1},${y1} ${x2},${y1} ${x2},${yBottom} ${x1},${yBottom}`,
      color,
    });
  }

  // Time labels on x-axis
  const numLabels = Math.min(6, sorted.length);
  const labelStep = Math.max(1, Math.floor(sorted.length / numLabels));
  const xLabels: { x: number; label: string }[] = [];
  for (let i = 0; i < sorted.length; i += labelStep) {
    const d = new Date(sorted[i].eval_at);
    xLabels.push({
      x: x(times[i]),
      label: d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    });
  }

  return (
    <div className="chart-container">
      <svg viewBox={`0 0 ${W} ${H}`} width="100%" preserveAspectRatio="xMidYMid meet">
        {/* Grid lines */}
        <line x1={PAD_L} y1={PAD_T + 20} x2={PAD_L + plotW} y2={PAD_T + 20} stroke="#21262d" strokeWidth="1" />
        <line x1={PAD_L} y1={PAD_T + plotH - 20} x2={PAD_L + plotW} y2={PAD_T + plotH - 20} stroke="#21262d" strokeWidth="1" />
        <line x1={PAD_L} y1={PAD_T + plotH} x2={PAD_L + plotW} y2={PAD_T + plotH} stroke="#30363d" strokeWidth="1" />

        {/* Y labels */}
        <text x={PAD_L - 8} y={PAD_T + 24} textAnchor="end" fill="#8b949e" fontSize="11">Pass</text>
        <text x={PAD_L - 8} y={PAD_T + plotH - 16} textAnchor="end" fill="#8b949e" fontSize="11">Fail</text>

        {/* Area segments */}
        {segments.map((seg, i) => (
          <polygon key={i} points={seg.points} fill={seg.color} opacity="0.25" />
        ))}

        {/* Line connecting points */}
        <polyline
          fill="none"
          stroke="#58a6ff"
          strokeWidth="1.5"
          points={sorted.map((h, i) => `${x(times[i])},${y(h.result)}`).join(' ')}
        />

        {/* Data points */}
        {sorted.map((h, i) => (
          <circle
            key={h.id}
            cx={x(times[i])}
            cy={y(h.result)}
            r="3"
            fill={resultColor(h.result)}
          >
            <title>{`${new Date(h.eval_at).toLocaleString()}\n${h.result} — ${h.state}\n${h.details}`}</title>
          </circle>
        ))}

        {/* X labels */}
        {xLabels.map((lbl, i) => (
          <text key={i} x={lbl.x} y={H - 8} textAnchor="middle" fill="#8b949e" fontSize="10">
            {lbl.label}
          </text>
        ))}
      </svg>
    </div>
  );
}

/* ── State Changes Table ── */
function StateChangesTable({ history }: { history: EvalHistory[] }) {
  // history is newest-first; find state transitions
  const sorted = [...history].reverse();
  const changes: EvalHistory[] = [];
  for (let i = 0; i < sorted.length; i++) {
    if (i === 0 || sorted[i].state !== sorted[i - 1].state) {
      changes.push(sorted[i]);
    }
  }
  // Show newest-first
  changes.reverse();

  if (changes.length === 0) return <p className="muted">No state changes yet.</p>;

  return (
    <table className="table">
      <thead>
        <tr>
          <th>Status</th>
          <th>DateTime</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        {changes.map(h => (
          <tr key={h.id}>
            <td><span className={`badge badge-${h.state}`}>{h.state}</span></td>
            <td>{new Date(h.eval_at).toLocaleString()}</td>
            <td style={{ fontSize: '0.8rem', color: '#8b949e' }}>{h.details}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/* ── Main Component ── */
export default function SwitchDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [sw, setSw] = useState<Switch | null>(null);
  const [history, setHistory] = useState<EvalHistory[]>([]);
  const [uptimePct, setUptimePct] = useState(100);
  const [totalEvals, setTotalEvals] = useState(0);
  const [passCount, setPassCount] = useState(0);
  const [failCount, setFailCount] = useState(0);
  const [range, setRange] = useState('1h');
  const [error, setError] = useState('');

  const load = () => {
    const swId = Number(id);
    api.getSwitch(swId).then(setSw).catch(e => setError(e.message));
    api.getSwitchHistory(swId, range).then(res => {
      setHistory(res.history || []);
      setUptimePct(res.uptime_pct);
      setTotalEvals(res.total_evals);
      setPassCount(res.pass_count);
      setFailCount(res.fail_count);
    }).catch(() => {});
  };

  useEffect(() => {
    load();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, [id, range]);

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

      {/* ── Uptime Summary Bar ── */}
      <div style={{ marginTop: '2rem' }}>
        <UptimeBar history={history} uptimePct={uptimePct} />
      </div>

      {/* ── Range Selector ── */}
      <div className="range-selector">
        {RANGES.map(r => (
          <button
            key={r.value}
            className={`range-btn ${range === r.value ? 'range-btn-active' : ''}`}
            onClick={() => setRange(r.value)}
          >
            {r.label}
          </button>
        ))}
        <span className="muted" style={{ marginLeft: 'auto', fontSize: '0.8rem' }}>
          {totalEvals} evals &middot; {passCount} pass &middot; {failCount} fail
        </span>
      </div>

      {/* ── Status Chart ── */}
      <StatusChart history={history} />

      {/* ── State Changes Table ── */}
      <h2 style={{ marginTop: '1.5rem', marginBottom: '0.5rem' }}>State Changes</h2>
      <StateChangesTable history={history} />
    </div>
  );
}
