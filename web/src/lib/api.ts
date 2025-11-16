import {
  Customer,
  CreateCustomerDTO,
  Reward,
  CreateRewardDTO,
  Rule,
  CreateRuleDTO,
  Campaign,
  CreateCampaignDTO,
  Budget,
  CreateBudgetDTO,
  TopupBudgetDTO,
  Issuance,
  Event,
  CreateEventDTO,
  LedgerEntry,
  LoginResponse,
  DashboardStats,
  PaginatedResponse,
} from './types';

const API_BASE = import.meta.env.VITE_API_URL || '/v1';

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public details?: any
  ) {
    super(message);
    this.name = 'APIError';
  }
}

export class APIClient {
  private token: string | null = null;
  private tenantId: string | null = null;

  constructor() {
    // Load token from localStorage on init
    const stored = localStorage.getItem('auth_token');
    if (stored) {
      this.token = stored;
    }
    const storedTenantId = localStorage.getItem('tenant_id');
    if (storedTenantId) {
      this.tenantId = storedTenantId;
    }
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('auth_token', token);
  }

  setTenantId(tenantId: string) {
    this.tenantId = tenantId;
    localStorage.setItem('tenant_id', tenantId);
  }

  clearAuth() {
    this.token = null;
    this.tenantId = null;
    localStorage.removeItem('auth_token');
    localStorage.removeItem('tenant_id');
  }

  getTenantId(): string {
    if (!this.tenantId) {
      throw new Error('Tenant ID not set');
    }
    return this.tenantId;
  }

  async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options?.headers as Record<string, string>) || {}),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      let errorMessage = `API error: ${response.statusText}`;
      let errorDetails;

      try {
        const errorData = await response.json();
        errorMessage = errorData.error || errorMessage;
        errorDetails = errorData.details;
      } catch {
        // If response is not JSON, use status text
      }

      throw new APIError(errorMessage, response.status, errorDetails);
    }

    return response.json();
  }

  // Auth endpoints
  auth = {
    login: (email: string, password: string) =>
      this.request<LoginResponse>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    me: () =>
      this.request<{ id: string; email: string; full_name: string; role: string; tenant_id: string }>('/auth/me'),
    refresh: (refreshToken: string) =>
      this.request<{ access_token: string; refresh_token: string; expires_in: number }>('/auth/refresh', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      }),
  };

  // Customer endpoints
  customers = {
    list: async (limit = 50, offset = 0) => {
      const response = await this.request<{ customers: Customer[]; total: number }>(
        `/tenants/${this.getTenantId()}/customers?limit=${limit}&offset=${offset}`
      );
      return { data: response.customers, total: response.total, page: 0, limit };
    },
    get: (id: string) =>
      this.request<Customer>(`/tenants/${this.getTenantId()}/customers/${id}`),
    create: (data: CreateCustomerDTO) =>
      this.request<Customer>(`/tenants/${this.getTenantId()}/customers`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    updateStatus: (id: string, status: 'active' | 'suspended' | 'deleted') =>
      this.request<Customer>(`/tenants/${this.getTenantId()}/customers/${id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      }),
  };

  // Reward endpoints
  rewards = {
    list: async () => {
      const response = await this.request<{ rewards: Reward[] }>(`/tenants/${this.getTenantId()}/reward-catalog`);
      return response.rewards;
    },
    get: (id: string) =>
      this.request<Reward>(`/tenants/${this.getTenantId()}/reward-catalog/${id}`),
    create: (data: CreateRewardDTO) =>
      this.request<Reward>(`/tenants/${this.getTenantId()}/reward-catalog`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<CreateRewardDTO>) =>
      this.request<Reward>(`/tenants/${this.getTenantId()}/reward-catalog/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
      }),
    uploadCodes: (id: string, codes: string[]) =>
      this.request<{ uploaded: number }>(`/tenants/${this.getTenantId()}/reward-catalog/${id}/upload-codes`, {
        method: 'POST',
        body: JSON.stringify({ codes }),
      }),
  };

  // Rule endpoints
  rules = {
    list: async () => {
      const response = await this.request<{ rules: Rule[] }>(`/tenants/${this.getTenantId()}/rules`);
      return response.rules;
    },
    get: (id: string) =>
      this.request<Rule>(`/tenants/${this.getTenantId()}/rules/${id}`),
    create: (data: CreateRuleDTO) =>
      this.request<Rule>(`/tenants/${this.getTenantId()}/rules`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<CreateRuleDTO>) =>
      this.request<Rule>(`/tenants/${this.getTenantId()}/rules/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      this.request<void>(`/tenants/${this.getTenantId()}/rules/${id}`, {
        method: 'DELETE',
      }),
  };

  // Campaign endpoints
  campaigns = {
    list: async () => {
      const response = await this.request<{ campaigns: Campaign[] }>(`/tenants/${this.getTenantId()}/campaigns`);
      return response.campaigns;
    },
    get: (id: string) =>
      this.request<Campaign>(`/tenants/${this.getTenantId()}/campaigns/${id}`),
    create: (data: CreateCampaignDTO) =>
      this.request<Campaign>(`/tenants/${this.getTenantId()}/campaigns`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: Partial<CreateCampaignDTO>) =>
      this.request<Campaign>(`/tenants/${this.getTenantId()}/campaigns/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
      }),
  };

  // Budget endpoints
  budgets = {
    list: async () => {
      const response = await this.request<{ budgets: Budget[] }>(`/tenants/${this.getTenantId()}/budgets`);
      return response.budgets;
    },
    get: (id: string) =>
      this.request<Budget>(`/tenants/${this.getTenantId()}/budgets/${id}`),
    create: (data: CreateBudgetDTO) =>
      this.request<Budget>(`/tenants/${this.getTenantId()}/budgets`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    topup: (id: string, data: TopupBudgetDTO) =>
      this.request<Budget>(`/tenants/${this.getTenantId()}/budgets/${id}/topup`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  };

  // Ledger endpoints
  ledger = {
    list: async (budgetId?: string) => {
      const query = budgetId ? `?budget_id=${budgetId}` : '';
      const response = await this.request<{ entries: LedgerEntry[] }>(`/tenants/${this.getTenantId()}/ledger${query}`);
      return response.entries;
    },
  };

  // Issuance endpoints
  issuances = {
    list: async (customerId?: string) => {
      const query = customerId ? `?customer_id=${customerId}` : '';
      const response = await this.request<{ issuances: Issuance[] }>(`/tenants/${this.getTenantId()}/issuances${query}`);
      return response.issuances;
    },
    get: (id: string) =>
      this.request<Issuance>(`/tenants/${this.getTenantId()}/issuances/${id}`),
    redeem: (id: string, otp?: string) =>
      this.request<Issuance>(`/tenants/${this.getTenantId()}/issuances/${id}/redeem`, {
        method: 'POST',
        body: JSON.stringify({ otp }),
      }),
    cancel: (id: string) =>
      this.request<Issuance>(`/tenants/${this.getTenantId()}/issuances/${id}/cancel`, {
        method: 'POST',
      }),
  };

  // Event endpoints
  events = {
    list: (limit = 50, offset = 0) =>
      this.request<PaginatedResponse<Event>>(
        `/tenants/${this.getTenantId()}/events?limit=${limit}&offset=${offset}`
      ),
    get: (id: string) =>
      this.request<Event>(`/tenants/${this.getTenantId()}/events/${id}`),
    create: (data: CreateEventDTO) =>
      this.request<Event>(`/tenants/${this.getTenantId()}/events`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  };

  // Analytics endpoints
  analytics = {
    getDashboardStats: () =>
      this.request<DashboardStats>(`/tenants/${this.getTenantId()}/analytics/dashboard`),
  };
}

export const api = new APIClient();
