import { z } from 'zod';

// E.164 phone number validation
const e164PhoneSchema = z
  .string()
  .regex(/^\+\d{1,15}$/, 'Must be a valid E.164 phone number (e.g., +263712345678)')
  .optional()
  .or(z.literal(''));

// Customer schemas
export const createCustomerSchema = z.object({
  phone_e164: e164PhoneSchema,
  external_ref: z.string().min(1, 'External reference is required').max(100),
});

export const updateCustomerSchema = z.object({
  phone_e164: e164PhoneSchema,
  external_ref: z.string().min(1).max(100).optional(),
  status: z.enum(['active', 'suspended', 'deleted']).optional(),
});

// Reward schemas
export const createRewardSchema = z.object({
  name: z.string().min(1, 'Reward name is required').max(200),
  type: z.enum([
    'discount',
    'voucher_code',
    'points_credit',
    'external_voucher',
    'physical_item',
    'webhook_custom',
  ]),
  face_value: z.coerce.number().positive().optional().or(z.literal('')),
  currency: z.enum(['ZWG', 'USD']).optional(),
  expiry_days: z.coerce.number().int().positive().optional().or(z.literal('')),
  inventory_total: z.coerce.number().int().positive().optional().or(z.literal('')),
  config: z.string().optional(),
});

export const uploadVoucherCodesSchema = z.object({
  codes: z.array(z.string()).min(1, 'At least one voucher code is required'),
});

// Rule schemas
export const createRuleSchema = z.object({
  name: z.string().min(1, 'Rule name is required').max(200),
  event_type: z.string().min(1, 'Event type is required'),
  conditions: z.any(), // JsonLogic object
  reward_id: z.string().uuid('Invalid reward ID'),
  per_user_cap: z.coerce.number().int().positive().default(1),
  global_cap: z.coerce.number().int().positive().optional().or(z.literal('')),
  cool_down_sec: z.coerce.number().int().nonnegative().default(0),
  campaign_id: z.string().uuid().optional().or(z.literal('')),
  priority: z.coerce.number().int().default(100),
});

// Campaign schemas
export const createCampaignSchema = z.object({
  name: z.string().min(1, 'Campaign name is required').max(200),
  start_date: z.string().min(1, 'Start date is required'),
  end_date: z.string().min(1, 'End date is required'),
  budget_id: z.string().uuid().optional().or(z.literal('')),
  description: z.string().optional(),
});

// Budget schemas
export const createBudgetSchema = z.object({
  name: z.string().min(1, 'Budget name is required').max(200),
  budget_type: z.enum(['campaign', 'reward', 'tenant']),
  currency: z.enum(['ZWG', 'USD']),
  cap_hard: z.coerce.number().positive('Hard cap must be positive'),
  cap_soft: z.coerce.number().positive().optional().or(z.literal('')),
  period_type: z.enum(['one_time', 'monthly', 'quarterly', 'yearly']),
  reset_day: z.coerce.number().int().min(1).max(31).optional().or(z.literal('')),
});

export const topupBudgetSchema = z.object({
  amount: z.coerce.number().positive('Amount must be positive'),
  notes: z.string().max(500).optional(),
});

// Event schemas
export const createEventSchema = z.object({
  customer_id: z.string().uuid('Invalid customer ID'),
  event_type: z.string().min(1, 'Event type is required'),
  event_data: z.string(), // JSON string
  idempotency_key: z.string().optional(),
});

// Type inference
export type CreateCustomerInput = z.infer<typeof createCustomerSchema>;
export type UpdateCustomerInput = z.infer<typeof updateCustomerSchema>;
export type CreateRewardInput = z.infer<typeof createRewardSchema>;
export type UploadVoucherCodesInput = z.infer<typeof uploadVoucherCodesSchema>;
export type CreateRuleInput = z.infer<typeof createRuleSchema>;
export type CreateCampaignInput = z.infer<typeof createCampaignSchema>;
export type CreateBudgetInput = z.infer<typeof createBudgetSchema>;
export type TopupBudgetInput = z.infer<typeof topupBudgetSchema>;
export type CreateEventInput = z.infer<typeof createEventSchema>;
