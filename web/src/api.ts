const BASE = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  if (res.status === 204) return {} as T;
  return res.json();
}

// Types
export interface Switch {
  id: number;
  name: string;
  signal: string;
  query: string;
  mode: string;
  state: string;
  auto_created: boolean;
  interval_seconds: number;
  grace_seconds: number;
  window_start: string;
  window_end: string;
  window_tz: string;
  min_samples: number;
  tolerance_multiplier: number;
  last_signal_at: string | null;
  next_expected_at: string | null;
  state_changed_at: string;
  eval_pass_count: number;
  eval_fail_count: number;
  created_at: string;
  updated_at: string;
}

export interface Dashboard {
  total: number;
  up: number;
  down: number;
  grace: number;
  learning: number;
  paused: number;
  new: number;
  switches: Switch[];
}

export interface EvalHistory {
  id: number;
  switch_id: number;
  eval_at: string;
  result: string;
  state: string;
  signal_at: string | null;
  details: string;
}

export interface AutoRule {
  id: number;
  loki_selector: string;
  pattern: string;
  active: boolean;
  min_samples: number;
  tolerance_multiplier: number;
  last_scan_at: string | null;
  created_at: string;
}

export interface AutoRulesResponse {
  rules: AutoRule[];
  auto_switch_count: number;
  learning_count: number;
}

// API calls
export const api = {
  dashboard: () => request<Dashboard>('/dashboard'),

  listSwitches: () => request<Switch[]>('/switches'),
  getSwitch: (id: number) => request<Switch>(`/switches/${id}`),
  createSwitch: (data: Partial<Switch>) => request<Switch>('/switches', { method: 'POST', body: JSON.stringify(data) }),
  updateSwitch: (id: number, data: Partial<Switch>) => request<Switch>(`/switches/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteSwitch: (id: number) => request<void>(`/switches/${id}`, { method: 'DELETE' }),
  pauseSwitch: (id: number) => request<void>(`/switches/${id}/pause`, { method: 'POST' }),
  resumeSwitch: (id: number) => request<void>(`/switches/${id}/resume`, { method: 'POST' }),
  getSwitchHistory: (id: number, limit = 50) => request<EvalHistory[]>(`/switches/${id}/history?limit=${limit}`),
  testQuery: (signal: string, query: string) => request<Record<string, any>>('/switches/test-query', { method: 'POST', body: JSON.stringify({ signal, query }) }),

  listAutoRules: () => request<AutoRulesResponse>('/auto-rules'),
  createAutoRule: (data: Partial<AutoRule>) => request<AutoRule>('/auto-rules', { method: 'POST', body: JSON.stringify(data) }),
  updateAutoRule: (id: number, data: Partial<AutoRule>) => request<AutoRule>(`/auto-rules/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteAutoRule: (id: number) => request<void>(`/auto-rules/${id}`, { method: 'DELETE' }),
};
