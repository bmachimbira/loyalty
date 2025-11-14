import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { api } from '@/lib/api';
import { Budget } from '@/lib/types';
import { toast } from 'sonner';
import { Plus, Wallet, RefreshCw, TrendingUp } from 'lucide-react';
import { CreateBudgetDialog } from '@/components/budgets/CreateBudgetDialog';
import { TopupBudgetDialog } from '@/components/budgets/TopupBudgetDialog';
import { Empty } from '@/components/Empty';
import { formatCurrency } from '@/lib/format';

export default function Budgets() {
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [topupBudget, setTopupBudget] = useState<Budget | null>(null);

  useEffect(() => {
    loadBudgets();
  }, []);

  const loadBudgets = async () => {
    setIsLoading(true);
    try {
      const data = await api.budgets.list();
      setBudgets(data);
    } catch (error: any) {
      toast.error('Failed to load budgets');
      console.error(error);
    } finally {
      setIsLoading(false);
    }
  };

  const calculateBalance = (budget: Budget) => {
    return budget.cap_hard - budget.balance_reserved - budget.balance_charged;
  };

  const getUtilizationPercent = (budget: Budget) => {
    const used = budget.balance_reserved + budget.balance_charged;
    return ((used / budget.cap_hard) * 100).toFixed(1);
  };

  const getUtilizationColor = (percent: number) => {
    if (percent >= 90) return 'text-red-600';
    if (percent >= 70) return 'text-yellow-600';
    return 'text-green-600';
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Budgets</h1>
          <p className="text-slate-600 mt-1">Manage your reward budgets and spending</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadBudgets}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Budget
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : budgets.length === 0 ? (
        <Empty
          icon={Wallet}
          title="No budgets yet"
          description="Create budgets to control reward spending"
          action={{
            label: 'Create Budget',
            onClick: () => setShowCreateDialog(true),
          }}
        />
      ) : (
        <div className="bg-white rounded-lg border border-slate-200 overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Hard Cap</TableHead>
                <TableHead>Reserved</TableHead>
                <TableHead>Charged</TableHead>
                <TableHead>Available</TableHead>
                <TableHead>Utilization</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {budgets.map((budget) => {
                const utilization = Number(getUtilizationPercent(budget));
                return (
                  <TableRow key={budget.id}>
                    <TableCell className="font-medium">{budget.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{budget.budget_type}</Badge>
                    </TableCell>
                    <TableCell>
                      {formatCurrency(budget.cap_hard, budget.currency)}
                    </TableCell>
                    <TableCell>
                      {formatCurrency(budget.balance_reserved, budget.currency)}
                    </TableCell>
                    <TableCell>
                      {formatCurrency(budget.balance_charged, budget.currency)}
                    </TableCell>
                    <TableCell>
                      {formatCurrency(calculateBalance(budget), budget.currency)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <TrendingUp
                          className={`h-4 w-4 ${getUtilizationColor(utilization)}`}
                        />
                        <span className={getUtilizationColor(utilization)}>
                          {utilization}%
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={budget.active ? 'default' : 'secondary'}>
                        {budget.active ? 'Active' : 'Inactive'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setTopupBudget(budget)}
                      >
                        Topup
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <CreateBudgetDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={loadBudgets}
      />

      <TopupBudgetDialog
        budget={topupBudget}
        open={!!topupBudget}
        onOpenChange={(open) => !open && setTopupBudget(null)}
        onSuccess={loadBudgets}
      />
    </div>
  );
}
