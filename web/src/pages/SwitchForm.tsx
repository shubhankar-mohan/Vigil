import { useEffect, useState, useRef } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api';

// #2 — Human-readable duration
function formatSeconds(s: number): string {
  if (s <= 0) return '';
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const parts = [];
  if (d) parts.push(`${d}d`);
  if (h) parts.push(`${h}h`);
  if (m) parts.push(`${m}m`);
  if (sec && !d && !h) parts.push(`${sec}s`);
  return parts.length ? `= ${parts.join(' ')}` : '';
}

// Preset intervals
const INTERVAL_PRESETS = [
  { label: '5m', value: 300 },
  { label: '15m', value: 900 },
  { label: '30m', value: 1800 },
  { label: '1h', value: 3600 },
  { label: '6h', value: 21600 },
  { label: '12h', value: 43200 },
  { label: '1d', value: 86400 },
];

const GRACE_PRESETS = [
  { label: '1m', value: 60 },
  { label: '5m', value: 300 },
  { label: '10m', value: 600 },
  { label: '30m', value: 1800 },
  { label: '1h', value: 3600 },
];

// #4 — Common timezones
const TIMEZONES = [
  'Asia/Kolkata', 'UTC', 'US/Eastern', 'US/Central', 'US/Pacific',
  'Europe/London', 'Europe/Berlin', 'Europe/Paris', 'Asia/Tokyo',
  'Asia/Shanghai', 'Asia/Singapore', 'Asia/Dubai', 'Australia/Sydney',
  'Pacific/Auckland', 'America/Sao_Paulo', 'Africa/Lagos',
];

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

  // #5 — Field validation
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [submitted, setSubmitted] = useState(false);

  // #4 — Timezone dropdown
  const [tzOpen, setTzOpen] = useState(false);
  const [tzFilter, setTzFilter] = useState('');
  const tzRef = useRef<HTMLDivElement>(null);

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

  // Close tz dropdown on outside click
  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (tzRef.current && !tzRef.current.contains(e.target as Node)) setTzOpen(false);
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const set = (field: string, value: string | number) => {
    setForm(f => ({ ...f, [field]: value }));
    setTestResult(null);
    if (submitted) setFieldErrors(e => ({ ...e, [field]: '' }));
  };

  // #5 — Validate form
  const validate = (): boolean => {
    const errors: Record<string, string> = {};
    if (!form.name.trim()) errors.name = 'Name is required';
    if (!form.query.trim()) errors.query = 'Query is required';
    if (form.mode === 'frequency') {
      if (form.interval_seconds <= 0) errors.interval_seconds = 'Must be greater than 0';
      if (form.grace_seconds < 0) errors.grace_seconds = 'Cannot be negative';
      if (form.window_start && !/^\d{1,2}:\d{2}$/.test(form.window_start)) errors.window_start = 'Use HH:MM format';
      if (form.window_end && !/^\d{1,2}:\d{2}$/.test(form.window_end)) errors.window_end = 'Use HH:MM format';
    }
    if (form.mode === 'irregularity') {
      if (form.min_samples < 3) errors.min_samples = 'Minimum 3 samples required';
      if (form.tolerance_multiplier <= 0) errors.tolerance_multiplier = 'Must be greater than 0';
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleTestQuery = async () => {
    if (!form.query) {
      setFieldErrors(e => ({ ...e, query: 'Enter a query first' }));
      return;
    }
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
    setSubmitted(true);
    setError('');
    if (!validate()) return;
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

  const filteredTz = TIMEZONES.filter(tz =>
    tz.toLowerCase().includes((tzFilter || form.window_tz).toLowerCase())
  );

  return (
    <div>
      {/* #8 — Breadcrumb */}
      <div className="breadcrumb">
        <Link to="/">Dashboard</Link>
        <span className="sep">/</span>
        <span>{isEdit ? 'Edit Switch' : 'New Switch'}</span>
      </div>

      <h1>{isEdit ? 'Edit Switch' : 'New Switch'}</h1>
      {/* #7 — Helper text */}
      <p className="page-subtitle">
        A switch monitors a signal (Prometheus metric or Loki log) and alerts when it stops arriving.
      </p>

      {error && <div className="error">{error}</div>}
      <form className="form" onSubmit={handleSubmit} noValidate>
        {/* Name — required */}
        <div className="form-group">
          <label>Name <span className="req">*</span></label>
          <input
            value={form.name}
            onChange={e => set('name', e.target.value)}
            placeholder="e.g. sync_awb_heartbeat"
            className={fieldErrors.name ? 'field-error' : ''}
          />
          {fieldErrors.name && <div className="field-error-msg">{fieldErrors.name}</div>}
        </div>

        <div className="form-row">
          <div className="form-group">
            <label>Signal Type <span className="req">*</span></label>
            <select value={form.signal} onChange={e => set('signal', e.target.value)}>
              <option value="prometheus">Prometheus Metric</option>
              <option value="loki">Loki Log Query</option>
            </select>
          </div>
          <div className="form-group">
            <label>Detection Mode <span className="req">*</span></label>
            <select value={form.mode} onChange={e => set('mode', e.target.value)}>
              <option value="frequency">Frequency</option>
              <option value="irregularity">Irregularity</option>
            </select>
          </div>
        </div>

        {/* Query — required, with test button */}
        <div className="form-group">
          <label>Query ({form.signal === 'prometheus' ? 'PromQL' : 'LogQL'}) <span className="req">*</span></label>
          <textarea
            value={form.query}
            onChange={e => set('query', e.target.value)}
            placeholder={form.signal === 'prometheus'
              ? 'cron_last_run_timestamp{cron_name="sync_awb"}'
              : '{job="diagonAlleyBE_prod"} |= "[CRON] sync_awb completed"'}
            className={fieldErrors.query ? 'field-error' : ''}
          />
          {fieldErrors.query && <div className="field-error-msg">{fieldErrors.query}</div>}
          {/* #3 — Prominent test button */}
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
            {/* #2 — Interval with presets and human-readable hint */}
            <div className="form-row">
              <div className="form-group">
                <label>Expected Every <span className="req">*</span></label>
                <input
                  type="number"
                  value={form.interval_seconds}
                  onChange={e => set('interval_seconds', parseInt(e.target.value) || 0)}
                  className={fieldErrors.interval_seconds ? 'field-error' : ''}
                />
                <div className="form-hint">
                  {formatSeconds(form.interval_seconds)}
                  {' '}
                  {INTERVAL_PRESETS.map(p => (
                    <button key={p.value} type="button" className="preset-btn" onClick={() => set('interval_seconds', p.value)}>
                      {p.label}
                    </button>
                  ))}
                </div>
                {fieldErrors.interval_seconds && <div className="field-error-msg">{fieldErrors.interval_seconds}</div>}
              </div>
              <div className="form-group">
                <label>Grace Period</label>
                <input
                  type="number"
                  value={form.grace_seconds}
                  onChange={e => set('grace_seconds', parseInt(e.target.value) || 0)}
                  className={fieldErrors.grace_seconds ? 'field-error' : ''}
                />
                <div className="form-hint">
                  {formatSeconds(form.grace_seconds)}
                  {' '}
                  {GRACE_PRESETS.map(p => (
                    <button key={p.value} type="button" className="preset-btn" onClick={() => set('grace_seconds', p.value)}>
                      {p.label}
                    </button>
                  ))}
                </div>
                {fieldErrors.grace_seconds && <div className="field-error-msg">{fieldErrors.grace_seconds}</div>}
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Window Start (optional, HH:MM)</label>
                <input
                  value={form.window_start}
                  onChange={e => set('window_start', e.target.value)}
                  placeholder="09:00"
                  className={fieldErrors.window_start ? 'field-error' : ''}
                />
                {fieldErrors.window_start && <div className="field-error-msg">{fieldErrors.window_start}</div>}
              </div>
              <div className="form-group">
                <label>Window End (optional, HH:MM)</label>
                <input
                  value={form.window_end}
                  onChange={e => set('window_end', e.target.value)}
                  placeholder="11:00"
                  className={fieldErrors.window_end ? 'field-error' : ''}
                />
                {fieldErrors.window_end && <div className="field-error-msg">{fieldErrors.window_end}</div>}
              </div>
            </div>
            {/* #4 — Timezone searchable dropdown */}
            <div className="form-group">
              <label>Timezone</label>
              <div className="tz-select" ref={tzRef}>
                <input
                  value={tzOpen ? tzFilter : form.window_tz}
                  onChange={e => { setTzFilter(e.target.value); setTzOpen(true); }}
                  onFocus={() => { setTzOpen(true); setTzFilter(''); }}
                  placeholder="Search timezone..."
                />
                {tzOpen && (
                  <div className="tz-dropdown">
                    {filteredTz.length === 0 ? (
                      <div className="tz-dropdown-item" style={{color:'#8b949e'}}>No match</div>
                    ) : (
                      filteredTz.map(tz => (
                        <div
                          key={tz}
                          className={`tz-dropdown-item ${tz === form.window_tz ? 'selected' : ''}`}
                          onClick={() => { set('window_tz', tz); setTzOpen(false); setTzFilter(''); }}
                        >
                          {tz}
                        </div>
                      ))
                    )}
                  </div>
                )}
              </div>
            </div>
          </>
        )}

        {form.mode === 'irregularity' && (
          <div className="form-row">
            <div className="form-group">
              <label>Min Samples <span className="req">*</span></label>
              <input
                type="number"
                value={form.min_samples}
                onChange={e => set('min_samples', parseInt(e.target.value) || 0)}
                className={fieldErrors.min_samples ? 'field-error' : ''}
              />
              {fieldErrors.min_samples && <div className="field-error-msg">{fieldErrors.min_samples}</div>}
            </div>
            <div className="form-group">
              <label>Tolerance Multiplier <span className="req">*</span></label>
              <input
                type="number"
                step="0.1"
                value={form.tolerance_multiplier}
                onChange={e => set('tolerance_multiplier', parseFloat(e.target.value) || 0)}
                className={fieldErrors.tolerance_multiplier ? 'field-error' : ''}
              />
              <div className="form-hint">e.g. 2.0 = alert if gap exceeds 2x the median interval</div>
              {fieldErrors.tolerance_multiplier && <div className="field-error-msg">{fieldErrors.tolerance_multiplier}</div>}
            </div>
          </div>
        )}

        {/* #6 — Create prominent, Cancel as ghost with separation */}
        <div className="btn-group">
          <button type="submit" className="btn btn-primary" disabled={saving}>
            {saving ? 'Saving...' : (isEdit ? 'Update Switch' : 'Create Switch')}
          </button>
          <button type="button" className="btn-ghost" onClick={() => navigate(-1)}>Cancel</button>
        </div>
      </form>
    </div>
  );
}
