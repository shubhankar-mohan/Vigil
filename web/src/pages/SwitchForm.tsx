import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { api } from '../api';

export default function SwitchForm() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEdit = !!id;

  const [form, setForm] = useState({
    name: '',
    signal: 'prometheus',
    query: '',
    mode: 'frequency',
    interval_seconds: 3600,
    grace_seconds: 300,
    window_start: '',
    window_end: '',
    window_tz: 'Asia/Kolkata',
    min_samples: 4,
    tolerance_multiplier: 2.0,
  });
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<Record<string, any> | null>(null);

  useEffect(() => {
    if (isEdit) {
      api.getSwitch(Number(id)).then(sw => {
        setForm({
          name: sw.name,
          signal: sw.signal,
          query: sw.query,
          mode: sw.mode,
          interval_seconds: sw.interval_seconds || 3600,
          grace_seconds: sw.grace_seconds || 300,
          window_start: sw.window_start || '',
          window_end: sw.window_end || '',
          window_tz: sw.window_tz || 'Asia/Kolkata',
          min_samples: sw.min_samples || 4,
          tolerance_multiplier: sw.tolerance_multiplier || 2.0,
        });
      }).catch(e => setError(e.message));
    }
  }, [id, isEdit]);

  const set = (field: string, value: string | number) => {
    setForm(f => ({ ...f, [field]: value }));
    setTestResult(null);
  };

  const handleTestQuery = async () => {
    if (!form.query) { setError('Enter a query first'); return; }
    setTesting(true);
    setTestResult(null);
    setError('');
    try {
      const result = await api.testQuery(form.signal, form.query);
      setTestResult(result);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setTesting(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      if (isEdit) {
        await api.updateSwitch(Number(id), form);
        navigate(`/switches/${id}`);
      } else {
        const sw = await api.createSwitch(form);
        navigate(`/switches/${sw.id}`);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <h1>{isEdit ? 'Edit Switch' : 'New Switch'}</h1>
      {error && <div className="error">{error}</div>}
      <form className="form" onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Name</label>
          <input value={form.name} onChange={e => set('name', e.target.value)} placeholder="e.g. sync_awb_heartbeat" required />
        </div>

        <div className="form-row">
          <div className="form-group">
            <label>Signal Type</label>
            <select value={form.signal} onChange={e => set('signal', e.target.value)}>
              <option value="prometheus">Prometheus Metric</option>
              <option value="loki">Loki Log Query</option>
            </select>
          </div>
          <div className="form-group">
            <label>Detection Mode</label>
            <select value={form.mode} onChange={e => set('mode', e.target.value)}>
              <option value="frequency">Frequency</option>
              <option value="irregularity">Irregularity</option>
            </select>
          </div>
        </div>

        <div className="form-group">
          <label>Query ({form.signal === 'prometheus' ? 'PromQL' : 'LogQL'})</label>
          <textarea
            value={form.query}
            onChange={e => set('query', e.target.value)}
            placeholder={form.signal === 'prometheus'
              ? 'cron_last_run_timestamp{cron_name="sync_awb"}'
              : '{job="diagon-alley"} |= "[CRON] sync_awb completed"'}
            required
          />
          <button
            type="button"
            className="btn btn-test"
            onClick={handleTestQuery}
            disabled={testing || !form.query}
            style={{ marginTop: '0.5rem' }}
          >
            {testing ? 'Testing...' : 'Test Query'}
          </button>
        </div>

        {testResult && (
          <div className={`test-result ${testResult.success ? 'test-success' : 'test-fail'}`}>
            <div className="test-result-header">
              {testResult.success ? 'Query OK' : 'Query Failed'}
            </div>
            {testResult.error && <div className="test-row"><span className="test-label">Error</span><span>{testResult.error}</span></div>}
            {testResult.message && <div className="test-row"><span className="test-label">Info</span><span>{testResult.message}</span></div>}
            {testResult.raw_value !== undefined && <div className="test-row"><span className="test-label">Raw Value</span><span className="test-mono">{testResult.raw_value}</span></div>}
            {testResult.signal_time && <div className="test-row"><span className="test-label">Signal Time</span><span className="test-mono">{testResult.signal_time}</span></div>}
            {testResult.signal_age && <div className="test-row"><span className="test-label">Signal Age</span><span className="test-mono">{testResult.signal_age}</span></div>}
            {testResult.last_occurrence && <div className="test-row"><span className="test-label">Last Match</span><span className="test-mono">{testResult.last_occurrence}</span></div>}
          </div>
        )}

        {form.mode === 'frequency' && (
          <>
            <div className="form-row">
              <div className="form-group">
                <label>Expected Every (seconds)</label>
                <input type="number" value={form.interval_seconds} onChange={e => set('interval_seconds', parseInt(e.target.value) || 0)} />
              </div>
              <div className="form-group">
                <label>Grace Period (seconds)</label>
                <input type="number" value={form.grace_seconds} onChange={e => set('grace_seconds', parseInt(e.target.value) || 0)} />
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Window Start (optional, HH:MM)</label>
                <input value={form.window_start} onChange={e => set('window_start', e.target.value)} placeholder="09:00" />
              </div>
              <div className="form-group">
                <label>Window End (optional, HH:MM)</label>
                <input value={form.window_end} onChange={e => set('window_end', e.target.value)} placeholder="11:00" />
              </div>
            </div>
            <div className="form-group">
              <label>Timezone</label>
              <input value={form.window_tz} onChange={e => set('window_tz', e.target.value)} />
            </div>
          </>
        )}

        {form.mode === 'irregularity' && (
          <div className="form-row">
            <div className="form-group">
              <label>Min Samples (before activating)</label>
              <input type="number" value={form.min_samples} onChange={e => set('min_samples', parseInt(e.target.value) || 0)} />
            </div>
            <div className="form-group">
              <label>Tolerance Multiplier (e.g. 2x median)</label>
              <input type="number" step="0.1" value={form.tolerance_multiplier} onChange={e => set('tolerance_multiplier', parseFloat(e.target.value) || 0)} />
            </div>
          </div>
        )}

        <div className="btn-group">
          <button type="submit" className="btn btn-primary" disabled={saving}>
            {saving ? 'Saving...' : (isEdit ? 'Update' : 'Create')}
          </button>
          <button type="button" className="btn" onClick={() => navigate(-1)}>Cancel</button>
        </div>
      </form>
    </div>
  );
}
