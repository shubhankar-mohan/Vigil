import { useEffect, useState } from 'react';
import { api } from '../api';
import type { AlertChannel, AlertLog } from '../api';

const CHANNEL_TYPES = [
  { value: 'slack', label: 'Slack', configHint: '{"webhook_url": "https://hooks.slack.com/services/..."}' },
  { value: 'webhook', label: 'Webhook', configHint: '{"url": "https://...", "method": "POST", "headers": {"Authorization": "Bearer ..."}}' },
  { value: 'discord', label: 'Discord', configHint: '{"webhook_url": "https://discord.com/api/webhooks/..."}' },
  { value: 'pagerduty', label: 'PagerDuty', configHint: '{"routing_key": "..."}' },
  { value: 'telegram', label: 'Telegram', configHint: '{"bot_token": "...", "chat_id": "..."}' },
];

function ChannelForm({
  initial,
  onSave,
  onCancel,
}: {
  initial?: AlertChannel;
  onSave: (data: { name: string; type: string; config: string; enabled: boolean }) => Promise<void>;
  onCancel: () => void;
}) {
  const [name, setName] = useState(initial?.name || '');
  const [type, setType] = useState(initial?.type || 'slack');
  const [config, setConfig] = useState(initial?.config || CHANNEL_TYPES[0].configHint);
  const [enabled, setEnabled] = useState(initial?.enabled ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!initial) {
      const ct = CHANNEL_TYPES.find(t => t.value === type);
      if (ct) setConfig(ct.configHint);
    }
  }, [type, initial]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      await onSave({ name, type, config, enabled });
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <form className="form" onSubmit={handleSubmit} style={{ marginBottom: '1.5rem' }}>
      {error && <div className="error">{error}</div>}
      <div className="form-row">
        <div className="form-group">
          <label>Name <span className="req">*</span></label>
          <input value={name} onChange={e => setName(e.target.value)} placeholder="my-slack-alerts" required />
        </div>
        <div className="form-group">
          <label>Type <span className="req">*</span></label>
          <select value={type} onChange={e => setType(e.target.value)}>
            {CHANNEL_TYPES.map(ct => (
              <option key={ct.value} value={ct.value}>{ct.label}</option>
            ))}
          </select>
        </div>
      </div>
      <div className="form-group">
        <label>Config (JSON) <span className="req">*</span></label>
        <textarea
          value={config}
          onChange={e => setConfig(e.target.value)}
          rows={4}
          style={{ fontFamily: 'monospace', fontSize: '0.82rem' }}
        />
        <div className="form-hint">
          {CHANNEL_TYPES.find(t => t.value === type)?.configHint}
        </div>
      </div>
      <div className="form-group">
        <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={enabled}
            onChange={e => setEnabled(e.target.checked)}
            style={{ width: 'auto' }}
          />
          Enabled
        </label>
      </div>
      <div className="btn-group">
        <button type="submit" className="btn btn-primary" disabled={saving}>
          {saving ? 'Saving...' : initial ? 'Update' : 'Create'}
        </button>
        <button type="button" className="btn-ghost" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}

export default function AlertChannels() {
  const [channels, setChannels] = useState<AlertChannel[]>([]);
  const [logs, setLogs] = useState<AlertLog[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<AlertChannel | null>(null);
  const [testing, setTesting] = useState<number | null>(null);
  const [testResult, setTestResult] = useState<{ id: number; success: boolean; message: string } | null>(null);

  const load = () => {
    api.listAlertChannels().then(setChannels).catch(() => {});
    api.listAlertLogs().then(setLogs).catch(() => {});
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async (data: { name: string; type: string; config: string; enabled: boolean }) => {
    await api.createAlertChannel(data);
    setShowForm(false);
    load();
  };

  const handleUpdate = async (data: { name: string; type: string; config: string; enabled: boolean }) => {
    if (!editing) return;
    await api.updateAlertChannel(editing.id, data);
    setEditing(null);
    load();
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this alert channel?')) return;
    await api.deleteAlertChannel(id);
    load();
  };

  const handleTest = async (id: number) => {
    setTesting(id);
    setTestResult(null);
    try {
      const res = await api.testAlertChannel(id);
      setTestResult({ id, success: res.success, message: res.message || res.error || '' });
    } catch (err: any) {
      setTestResult({ id, success: false, message: err.message });
    } finally {
      setTesting(null);
    }
  };

  const handleToggle = async (ch: AlertChannel) => {
    await api.updateAlertChannel(ch.id, { ...ch, enabled: !ch.enabled });
    load();
  };

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Alert Channels</h1>
          <p className="page-subtitle">Configure where Vigil sends notifications on state changes</p>
        </div>
        {!showForm && !editing && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Add Channel</button>
        )}
      </div>

      {showForm && (
        <ChannelForm onSave={handleCreate} onCancel={() => setShowForm(false)} />
      )}

      {editing && (
        <ChannelForm initial={editing} onSave={handleUpdate} onCancel={() => setEditing(null)} />
      )}

      {channels.length === 0 && !showForm ? (
        <p className="muted">No alert channels configured. Add one to receive notifications.</p>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {channels.map(ch => (
              <tr key={ch.id}>
                <td>{ch.name}</td>
                <td>
                  <span className="channel-type-badge">{ch.type}</span>
                </td>
                <td>
                  <button
                    className={`badge ${ch.enabled ? 'badge-up' : 'badge-paused'}`}
                    onClick={() => handleToggle(ch)}
                    style={{ cursor: 'pointer', border: 'none' }}
                    title="Click to toggle"
                  >
                    {ch.enabled ? 'enabled' : 'disabled'}
                  </button>
                </td>
                <td>
                  <div style={{ display: 'flex', gap: '0.4rem', alignItems: 'center' }}>
                    <button
                      className="btn btn-sm"
                      onClick={() => handleTest(ch.id)}
                      disabled={testing === ch.id}
                    >
                      {testing === ch.id ? 'Sending...' : 'Test'}
                    </button>
                    <button className="btn btn-sm" onClick={() => setEditing(ch)}>Edit</button>
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(ch.id)}>Delete</button>
                    {testResult?.id === ch.id && (
                      <span style={{ fontSize: '0.8rem', color: testResult.success ? '#40c057' : '#f03e3e' }}>
                        {testResult.success ? 'Sent!' : testResult.message}
                      </span>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2 style={{ marginTop: '2rem', marginBottom: '0.5rem' }}>Recent Alert Log</h2>
      {logs.length === 0 ? (
        <p className="muted">No alerts sent yet.</p>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Switch</th>
              <th>Transition</th>
              <th>Status</th>
              <th>Error</th>
            </tr>
          </thead>
          <tbody>
            {logs.map(l => (
              <tr key={l.id}>
                <td style={{ fontSize: '0.82rem' }}>{new Date(l.sent_at).toLocaleString()}</td>
                <td>{l.switch_name}</td>
                <td>
                  <span className={`badge badge-${l.old_state}`}>{l.old_state}</span>
                  {' → '}
                  <span className={`badge badge-${l.new_state}`}>{l.new_state}</span>
                </td>
                <td>
                  <span className={`badge ${l.success ? 'badge-up' : 'badge-down'}`}>
                    {l.success ? 'sent' : 'failed'}
                  </span>
                </td>
                <td style={{ fontSize: '0.78rem', color: '#8b949e' }}>{l.error || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
