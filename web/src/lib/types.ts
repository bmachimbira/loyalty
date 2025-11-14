// Core types matching backend database schema

export interface Customer {
  id: string;
  tenant_id: string;
  phone_e164: string | null;
  external_ref: string | null;
  status: 'active' | 'suspended' | 'deleted';
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerDTO {
  phone_e164?: string;
  external_ref?: string;
}

export interface Reward {
  id: string;
  tenant_id: string;
  name: string;
  type: 'discount' | 'voucher_code' | 'points_credit' | 'external_voucher' | 'physical_item' | 'webhook_custom';
  face_value: number | null;
  currency: 'ZWG' | 'USD' | null;
  expiry_days: number | null;
  active: boolean;
  inventory_total: number | null;
  inventory_used: number;
  config: Record<string, any> | null;
  created_at: string;
  updated_at: string;
}

export interface CreateRewardDTO {
  name: string;
  type: Reward['type'];
  face_value?: number;
  currency?: 'ZWG' | 'USD';
  expiry_days?: number;
  inventory_total?: number;
  config?: Record<string, any>;
  active?: boolean;
}

export interface Rule {
  id: string;
  tenant_id: string;
  name: string;
  event_type: string;
  conditions: any; // JsonLogic
  reward_id: string;
  per_user_cap: number;
  global_cap: number | null;
  cool_down_sec: number;
  active: boolean;
  created_at: string;
  updated_at: string;
  reward?: Reward;
}

export interface CreateRuleDTO {
  name: string;
  event_type: string;
  conditions: any;
  reward_id: string;
  per_user_cap?: number;
  global_cap?: number;
  cool_down_sec?: number;
  active?: boolean;
}

export interface Campaign {
  id: string;
  tenant_id: string;
  name: string;
  start_date: string;
  end_date: string;
  budget_id: string | null;
  active: boolean;
  created_at: string;
  updated_at: string;
  budget?: Budget;
}

export interface CreateCampaignDTO {
  name: string;
  start_date: string;
  end_date: string;
  budget_id?: string;
}

export interface Budget {
  id: string;
  tenant_id: string;
  name: string;
  budget_type: 'campaign' | 'reward' | 'tenant';
  currency: 'ZWG' | 'USD';
  cap_hard: number;
  cap_soft: number | null;
  balance_reserved: number;
  balance_charged: number;
  period_type: 'one_time' | 'monthly' | 'quarterly' | 'yearly';
  reset_day: number | null;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateBudgetDTO {
  name: string;
  budget_type: Budget['budget_type'];
  currency: 'ZWG' | 'USD';
  cap_hard: number;
  cap_soft?: number;
  period_type: Budget['period_type'];
  reset_day?: number;
}

export interface TopupBudgetDTO {
  amount: number;
  notes?: string;
}

export interface Issuance {
  id: string;
  tenant_id: string;
  customer_id: string;
  rule_id: string | null;
  reward_id: string;
  event_id: string | null;
  state: 'reserved' | 'issued' | 'redeemed' | 'expired' | 'cancelled';
  face_value: number | null;
  currency: 'ZWG' | 'USD' | null;
  code: string | null;
  otp: string | null;
  metadata: Record<string, any> | null;
  expires_at: string | null;
  issued_at: string | null;
  redeemed_at: string | null;
  created_at: string;
  updated_at: string;
  customer?: Customer;
  reward?: Reward;
  rule?: Rule;
}

export interface Event {
  id: string;
  tenant_id: string;
  customer_id: string;
  event_type: string;
  event_data: Record<string, any>;
  idempotency_key: string | null;
  processed: boolean;
  created_at: string;
  customer?: Customer;
}

export interface CreateEventDTO {
  customer_id: string;
  event_type: string;
  event_data: Record<string, any>;
  idempotency_key?: string;
}

export interface LedgerEntry {
  id: string;
  tenant_id: string;
  budget_id: string;
  entry_type: 'reserve' | 'charge' | 'release' | 'topup';
  amount: number;
  currency: 'ZWG' | 'USD';
  issuance_id: string | null;
  notes: string | null;
  created_at: string;
  budget?: Budget;
}

export interface User {
  id: string;
  tenant_id: string;
  email: string;
  role: 'admin' | 'staff';
  active: boolean;
  created_at: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface DashboardStats {
  active_customers: number;
  events_today: number;
  rewards_issued_today: number;
  redemption_rate: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}

export interface APIError {
  error: string;
  details?: any;
}
