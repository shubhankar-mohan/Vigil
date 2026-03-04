import { useEffect, useState } from 'react';
import { api } from '../api';
import type { AutoRule, AutoRulesResponse } from '../api';

export default function AutoRules() {
  const [data, setData] = useState<AutoRulesResponse | null>(null);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    loki_selector: '',
    pattern: '',
    min_samples: 4,
    tolerance_multiplier: 2.0,
  });

  const load = () => {
    api.listAutoRules().then(setData).catch(e => setError(e.message));
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createAutoRule(form);
      setForm({ loki_selector: '', pattern: '', min_samples: 4, tolerance_multiplier: 2.0 });
      setShowForm(false);
      load();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleToggle = async (rule: AutoRule) => {
    await api.updateAutoRule(rule.id, { active: !rule.active });
    load();
  };

  const handleDelete = async (rule: AutoRule) => {
    if (!confirm('Delete this auto-discovery rule?')) return;
    await api.deleteAutoRule(rule.id);
    load();
  };

  if (error) return <div className="error">{error}</div>;
  if (!data) return <p className="muted">Loading...</p>;

  return (
    <div>
      <div className="page-header">
        <h1>Auto-Discovery</h1>
        <button className="btn btn-primary" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : '+ Add Rule'}
        </button>
      </div>

      <div className="stats" style={{gridTemplateColumns: 'repeat(2, 1fr)'}}>
        <div className="stat-card">
          <div className="number">{data.auto_switch_count}</div>
          <div className="label">Auto-created Switches</div>
        </div>
        <div className="stat-card">
          <div className="number">{data.learning_count}</div>
          <div className="label">Still Learning</div>
        </div>
      </div>

      {showForm && (
        <form className="form" onSubmit={handleCreate} style={{marginBottom: '1.5rem'}}>
          <div className="form-group">
            <label>Loki Selector</label>
            <input
              value={form.loki_selector}
              onChange={e => setForm(f => ({...f, loki_selector: e.target.value}))}
              placeholder='{job="diagon-alley"}'
              required
            />
          </div>
          <div className="form-group">
            <label>Pattern Filter (optional, glob)</label>
            <input
              value={form.pattern}
              onChange={e => setForm(f => ({...f, pattern: e.target.value}))}
              placeholder='[CRON]*'
            />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>Min Samples</label>
              <input type="number" value={form.min_samples} onChange={e => setForm(f => ({...f, min_samples: parseInt(e.target.value) || 4}))} />
            </div>
            <div className="form-group">
              <label>Tolerance Multiplier</label>
              <input type="number" step="0.1" value={form.tolerance_multiplier} onChange={e => setForm(f => ({...f, tolerance_multiplier: parseFloat(e.target.value) || 2.0}))} />
            </div>
          </div>
          <button type="submit" className="btn btn-primary">Create Rule</button>
        </form>
      )}

      {data.rules.length === 0 ? (
        <p className="muted">No auto-discovery rules. Add one to automatically detect recurring log patterns.</p>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Loki Selector</th>
              <th>Pattern</th>
              <th>Status</th>
              <th>Last Scan</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {data.rules.map(rule => (
              <tr key={rule.id}>
                <td style={{fontFamily: 'monospace', fontSize: '0.85rem'}}>{rule.loki_selector}</td>
                <td>{rule.pattern || '*'}</td>
                <td>
                  <span className={`badge ${rule.active ? 'badge-up' : 'badge-paused'}`}>
                    {rule.active ? 'Active' : 'Paused'}
                  </span>
                </td>
                <td>{rule.last_scan_at ? new Date(rule.last_scan_at).toLocaleString() : 'Never'}</td>
                <td>
                  <div className="btn-group" style={{margin: 0}}>
                    <button className="btn btn-sm" onClick={() => handleToggle(rule)}>
                      {rule.active ? 'Pause' : 'Resume'}
                    </button>
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(rule)}>Delete</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
