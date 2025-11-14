import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { topupBudgetSchema, TopupBudgetInput } from '@/lib/schemas';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import { Budget } from '@/lib/types';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Loader2 } from 'lucide-react';
import { formatCurrency } from '@/lib/format';

interface TopupBudgetDialogProps {
  budget: Budget | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function TopupBudgetDialog({
  budget,
  open,
  onOpenChange,
  onSuccess,
}: TopupBudgetDialogProps) {
  const form = useForm({
    resolver: zodResolver(topupBudgetSchema),
    defaultValues: {
      amount: '' as any,
      notes: '',
    },
  });

  const onSubmit = async (data: TopupBudgetInput) => {
    if (!budget) return;

    try {
      await api.budgets.topup(budget.id, {
        amount: Number(data.amount),
        notes: data.notes,
      });
      toast.success('Budget topped up successfully');
      form.reset();
      onOpenChange(false);
      onSuccess();
    } catch (error: any) {
      toast.error(error.message || 'Failed to topup budget');
    }
  };

  const currentBalance = budget
    ? budget.cap_hard - budget.balance_reserved - budget.balance_charged
    : 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[450px]">
        <DialogHeader>
          <DialogTitle>Topup Budget</DialogTitle>
          <DialogDescription>
            Add funds to: <strong>{budget?.name}</strong>
          </DialogDescription>
        </DialogHeader>

        {budget && (
          <div className="bg-slate-50 p-4 rounded-lg space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-slate-600">Hard Cap:</span>
              <span className="font-medium">
                {formatCurrency(budget.cap_hard, budget.currency)}
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-600">Current Balance:</span>
              <span className="font-medium">
                {formatCurrency(currentBalance, budget.currency)}
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-600">Reserved:</span>
              <span className="font-medium">
                {formatCurrency(budget.balance_reserved, budget.currency)}
              </span>
            </div>
          </div>
        )}

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="amount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Topup Amount *</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="0.01"
                      placeholder="1000.00"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Amount to add to the budget in {budget?.currency}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="notes"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Notes</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="Optional notes about this topup..."
                      rows={3}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={form.formState.isSubmitting}>
                {form.formState.isSubmitting && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Topup Budget
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
