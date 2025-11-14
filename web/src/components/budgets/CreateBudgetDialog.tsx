import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { createBudgetSchema, CreateBudgetInput } from '@/lib/schemas';
import { api } from '@/lib/api';
import { toast } from 'sonner';
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
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Loader2 } from 'lucide-react';

interface CreateBudgetDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function CreateBudgetDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateBudgetDialogProps) {
  const form = useForm({
    resolver: zodResolver(createBudgetSchema),
    defaultValues: {
      name: '',
      budget_type: 'tenant' as const,
      currency: 'ZWG' as const,
      cap_hard: '' as any,
      cap_soft: '',
      period_type: 'one_time' as const,
      reset_day: '',
    },
  });

  const onSubmit = async (data: CreateBudgetInput) => {
    try {
      await api.budgets.create({
        name: data.name,
        budget_type: data.budget_type,
        currency: data.currency,
        cap_hard: Number(data.cap_hard),
        cap_soft: data.cap_soft ? Number(data.cap_soft) : undefined,
        period_type: data.period_type,
        reset_day: data.reset_day ? Number(data.reset_day) : undefined,
      });
      toast.success('Budget created successfully');
      form.reset();
      onOpenChange(false);
      onSuccess();
    } catch (error: any) {
      toast.error(error.message || 'Failed to create budget');
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Create Budget</DialogTitle>
          <DialogDescription>
            Set up a new budget to track and limit campaign spending
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Budget Name *</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., Q1 2025 Marketing Budget" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="budget_type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Budget Type *</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="tenant">Tenant</SelectItem>
                        <SelectItem value="campaign">Campaign</SelectItem>
                        <SelectItem value="reward">Reward</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="currency"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Currency *</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="ZWG">ZWG</SelectItem>
                        <SelectItem value="USD">USD</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="cap_hard"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Hard Cap *</FormLabel>
                    <FormControl>
                      <Input type="number" step="0.01" placeholder="10000.00" {...field} />
                    </FormControl>
                    <FormDescription>Maximum allowed</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="cap_soft"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Soft Cap</FormLabel>
                    <FormControl>
                      <Input type="number" step="0.01" placeholder="8000.00" {...field} />
                    </FormControl>
                    <FormDescription>Warning threshold</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="period_type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Period Type *</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="one_time">One Time</SelectItem>
                        <SelectItem value="monthly">Monthly</SelectItem>
                        <SelectItem value="quarterly">Quarterly</SelectItem>
                        <SelectItem value="yearly">Yearly</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="reset_day"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Reset Day</FormLabel>
                    <FormControl>
                      <Input type="number" min="1" max="31" placeholder="1" {...field} />
                    </FormControl>
                    <FormDescription>Day of month</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

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
                Create Budget
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
