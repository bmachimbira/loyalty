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
import { Plus, Wallet } from 'lucide-react';

export default function Budgets() {
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadBudgets();
  }, []);

  const loadBudgets = async () => {
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

  const formatCurrency = (amount: number, currency: string) => {
    return `${currency} ${amount.toFixed(2)}`;
  };

  const calculateBalance = (budget: Budget) => {
    return budget.cap_hard - budget.balance_reserved - budget.balance_charged;
  };

  const getUtilizationPercent = (budget: Budget) => {
    const used = budget.balance_reserved + budget.balance_charged;
    return ((used / budget.cap_hard) * 100).toFixed(1);
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Budgets</h1>
          <p className="text-slate-600 mt-1">Manage your reward budgets and spending</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create Budget
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : budgets.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-slate-200">
          <Wallet className="h-12 w-12 text-slate-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-slate-900 mb-2">No budgets yet</h3>
          <p className="text-slate-600 mb-4">Create budgets to control reward spending</p>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Create Budget
          </Button>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-slate-200">
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
              {budgets.map((budget) => (
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
                  <TableCell>{getUtilizationPercent(budget)}%</TableCell>
                  <TableCell>
                    <Badge variant={budget.active ? 'default' : 'secondary'}>
                      {budget.active ? 'Active' : 'Inactive'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button variant="ghost" size="sm">
                        Topup
                      </Button>
                      <Button variant="ghost" size="sm">
                        View
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
