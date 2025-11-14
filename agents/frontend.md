# Frontend Agent

## Mission
Build the merchant console using React, TypeScript, Tailwind CSS, and shadcn/ui components.

## Prerequisites
- Node.js 18+
- Understanding of React, TypeScript
- Familiarity with shadcn/ui component library
- Review spec: `Zimbabwe-White-Label-Loyalty-Spec-v1.0.md`

## Tasks

### 1. Setup shadcn/ui Components

Initialize shadcn/ui:
```bash
cd web
npx shadcn-ui@latest init
```

Install required components:
```bash
npx shadcn-ui@latest add button
npx shadcn-ui@latest add card
npx shadcn-ui@latest add input
npx shadcn-ui@latest add table
npx shadcn-ui@latest add dialog
npx shadcn-ui@latest add form
npx shadcn-ui@latest add select
npx shadcn-ui@latest add tabs
npx shadcn-ui@latest add badge
npx shadcn-ui@latest add dropdown-menu
npx shadcn-ui@latest add toast
npx shadcn-ui@latest add alert
npx shadcn-ui@latest add skeleton
```

### 2. API Client

**File**: `web/src/lib/api.ts`

```typescript
const API_BASE = import.meta.env.VITE_API_URL || '/v1';

export class APIClient {
  private token: string | null = null;

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('auth_token', token);
  }

  async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const headers = {
      'Content-Type': 'application/json',
      ...(this.token && { Authorization: `Bearer ${this.token}` }),
      ...options?.headers,
    };

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.statusText}`);
    }

    return response.json();
  }

  // Customer endpoints
  customers = {
    list: (tenantId: string) =>
      this.request<Customer[]>(`/tenants/${tenantId}/customers`),
    get: (tenantId: string, id: string) =>
      this.request<Customer>(`/tenants/${tenantId}/customers/${id}`),
    create: (tenantId: string, data: CreateCustomerDTO) =>
      this.request<Customer>(`/tenants/${tenantId}/customers`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  };

  // Add more endpoints...
}

export const api = new APIClient();
```

**File**: `web/src/lib/types.ts`

```typescript
export interface Customer {
  id: string;
  tenant_id: string;
  phone_e164: string | null;
  external_ref: string | null;
  status: string;
  created_at: string;
}

export interface Reward {
  id: string;
  tenant_id: string;
  name: string;
  type: 'discount' | 'voucher_code' | 'points_credit' | 'external_voucher' | 'physical_item' | 'webhook_custom';
  face_value: number | null;
  currency: 'ZWG' | 'USD' | null;
  active: boolean;
}

export interface Rule {
  id: string;
  name: string;
  event_type: string;
  conditions: any; // JsonLogic
  reward_id: string;
  per_user_cap: number;
  global_cap: number | null;
  cool_down_sec: number;
  active: boolean;
}

// Add more types...
```

### 3. Authentication

**File**: `web/src/contexts/AuthContext.tsx`

```typescript
import React, { createContext, useContext, useState, useEffect } from 'react';
import { api } from '@/lib/api';

interface AuthContextType {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    // Check for stored token
    const token = localStorage.getItem('auth_token');
    if (token) {
      api.setToken(token);
      // Fetch user profile
    }
  }, []);

  const login = async (email: string, password: string) => {
    // Call login API
    const { token, user } = await api.auth.login(email, password);
    api.setToken(token);
    setUser(user);
  };

  const logout = () => {
    localStorage.removeItem('auth_token');
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, login, logout, isAuthenticated: !!user }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
};
```

### 4. Layout Components

**File**: `web/src/components/Layout.tsx`

```typescript
import { Outlet, Link } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';

export function Layout() {
  const { user, logout } = useAuth();

  return (
    <div className="min-h-screen bg-background">
      <nav className="border-b">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <div className="flex gap-6">
            <Link to="/" className="font-bold text-xl">Loyalty Platform</Link>
            <Link to="/customers">Customers</Link>
            <Link to="/rewards">Rewards</Link>
            <Link to="/rules">Rules</Link>
            <Link to="/campaigns">Campaigns</Link>
            <Link to="/budgets">Budgets</Link>
          </div>
          <div>
            <span className="mr-4">{user?.email}</span>
            <Button onClick={logout}>Logout</Button>
          </div>
        </div>
      </nav>
      <main className="container mx-auto p-6">
        <Outlet />
      </main>
    </div>
  );
}
```

### 5. Dashboard Page

**File**: `web/src/pages/Dashboard.tsx`

```typescript
import { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { api } from '@/lib/api';

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);

  useEffect(() => {
    // Fetch dashboard stats
    api.analytics.getDashboardStats().then(setStats);
  }, []);

  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Active Customers
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{stats?.active_customers || 0}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Events Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{stats?.events_today || 0}</p>
          </CardContent>
        </Card>

        {/* Add more cards */}
      </div>

      {/* Recent activity, charts, etc. */}
    </div>
  );
}
```

### 6. Customers Page

**File**: `web/src/pages/Customers.tsx`

```typescript
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Table } from '@/components/ui/table';
import { Dialog } from '@/components/ui/dialog';
import { api } from '@/lib/api';
import { CreateCustomerForm } from '@/components/customers/CreateCustomerForm';

export default function Customers() {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  useEffect(() => {
    loadCustomers();
  }, []);

  const loadCustomers = async () => {
    const data = await api.customers.list(tenantId);
    setCustomers(data);
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Customers</h1>
        <Button onClick={() => setShowCreateDialog(true)}>
          Add Customer
        </Button>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Phone</TableHead>
            <TableHead>External Ref</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {customers.map((customer) => (
            <TableRow key={customer.id}>
              <TableCell>{customer.phone_e164}</TableCell>
              <TableCell>{customer.external_ref}</TableCell>
              <TableCell>
                <Badge>{customer.status}</Badge>
              </TableCell>
              <TableCell>{formatDate(customer.created_at)}</TableCell>
              <TableCell>
                <Button variant="ghost" size="sm">View</Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <CreateCustomerForm onSuccess={() => {
          setShowCreateDialog(false);
          loadCustomers();
        }} />
      </Dialog>
    </div>
  );
}
```

### 7. Rewards Page

**File**: `web/src/pages/Rewards.tsx`

Implement:
- List rewards with filters (active/inactive, by type)
- Create/edit reward dialog
- Upload voucher codes (CSV)
- Deactivate rewards

### 8. Rules Page

**File**: `web/src/pages/Rules.tsx`

Implement:
- List rules with status
- Create/edit rule dialog
- **Rule Builder**: Visual condition builder
- **Rule Simulator**: Test rules with sample events
- Activate/deactivate rules

**File**: `web/src/components/rules/RuleBuilder.tsx`

Visual builder for JsonLogic conditions:
- Event type selector
- Condition fields (amount, currency, etc.)
- Comparison operators
- Multiple conditions (AND/OR)

**File**: `web/src/components/rules/RuleSimulator.tsx`

Test rule with sample event:
```typescript
// Input: Event JSON
// Output: Would trigger? Reason if not
```

### 9. Campaigns Page

**File**: `web/src/pages/Campaigns.tsx`

Implement:
- List campaigns with date ranges
- Create/edit campaign
- Assign budget to campaign
- View campaign performance

### 10. Budgets Page

**File**: `web/src/pages/Budgets.tsx`

Implement:
- List budgets with balance/caps
- Create budget
- Topup budget
- View ledger entries
- Budget utilization chart

### 11. Forms

**File**: `web/src/components/customers/CreateCustomerForm.tsx`
**File**: `web/src/components/rewards/CreateRewardForm.tsx`
**File**: `web/src/components/rules/CreateRuleForm.tsx`
**File**: `web/src/components/campaigns/CreateCampaignForm.tsx`
**File**: `web/src/components/budgets/CreateBudgetForm.tsx`

Use React Hook Form with Zod validation.

### 12. Theming

**File**: `web/src/lib/theme.ts`

Support tenant-specific themes from `tenants.theme` JSON:
```typescript
export function applyTenantTheme(theme: TenantTheme) {
  const root = document.documentElement;
  root.style.setProperty('--primary', theme.colors.primary);
  root.style.setProperty('--secondary', theme.colors.secondary);
  // etc.
}
```

### 13. Error Handling

**File**: `web/src/components/ErrorBoundary.tsx`

Catch and display errors gracefully.

**File**: `web/src/lib/toast.ts`

Toast notifications for success/error messages.

## Completion Criteria

- [ ] All pages implemented with full CRUD
- [ ] Authentication working
- [ ] API client complete
- [ ] Forms with validation
- [ ] Tables with pagination
- [ ] Rule builder UI functional
- [ ] Rule simulator working
- [ ] Responsive design
- [ ] Error handling
- [ ] Loading states
- [ ] Tenant theming support
