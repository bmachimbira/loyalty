import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { createRewardSchema, CreateRewardInput } from '@/lib/schemas';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import { Reward } from '@/lib/types';
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
import { JsonEditor } from '@/components/JsonEditor';
import { useEffect } from 'react';

interface CreateRewardDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  reward?: Reward | null;
}

export function CreateRewardDialog({
  open,
  onOpenChange,
  onSuccess,
  reward,
}: CreateRewardDialogProps) {
  const isEdit = !!reward;

  const form = useForm({
    resolver: zodResolver(createRewardSchema),
    defaultValues: {
      name: '',
      type: 'discount' as const,
      face_value: '',
      currency: 'ZWG' as const,
      expiry_days: '',
      inventory_total: '',
      config: '{}',
    },
  });

  useEffect(() => {
    if (reward && open) {
      form.reset({
        name: reward.name,
        type: reward.type,
        face_value: reward.face_value?.toString() || '',
        currency: reward.currency || 'ZWG',
        expiry_days: reward.expiry_days?.toString() || '',
        inventory_total: reward.inventory_total?.toString() || '',
        config: JSON.stringify(reward.config || {}, null, 2),
      });
    } else if (!open) {
      form.reset();
    }
  }, [reward, open]);

  const onSubmit = async (data: CreateRewardInput) => {
    try {
      const payload = {
        name: data.name,
        type: data.type,
        face_value: data.face_value ? Number(data.face_value) : undefined,
        currency: data.currency,
        expiry_days: data.expiry_days ? Number(data.expiry_days) : undefined,
        inventory_total: data.inventory_total ? Number(data.inventory_total) : undefined,
        config: data.config ? JSON.parse(data.config) : undefined,
      };

      if (isEdit && reward) {
        await api.rewards.update(reward.id, payload);
        toast.success('Reward updated successfully');
      } else {
        await api.rewards.create(payload);
        toast.success('Reward created successfully');
      }

      form.reset();
      onOpenChange(false);
      onSuccess();
    } catch (error: any) {
      toast.error(error.message || `Failed to ${isEdit ? 'update' : 'create'} reward`);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Reward' : 'Create Reward'}</DialogTitle>
          <DialogDescription>
            {isEdit ? 'Update reward details' : 'Add a new reward to your catalog'}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Reward Name *</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., $5 Discount" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Reward Type *</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select type" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="discount">Discount</SelectItem>
                      <SelectItem value="voucher_code">Voucher Code</SelectItem>
                      <SelectItem value="points_credit">Points Credit</SelectItem>
                      <SelectItem value="external_voucher">External Voucher</SelectItem>
                      <SelectItem value="physical_item">Physical Item</SelectItem>
                      <SelectItem value="webhook_custom">Webhook Custom</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="face_value"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Face Value</FormLabel>
                    <FormControl>
                      <Input type="number" step="0.01" placeholder="0.00" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="currency"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Currency</FormLabel>
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
                name="expiry_days"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Expiry Days</FormLabel>
                    <FormControl>
                      <Input type="number" placeholder="30" {...field} />
                    </FormControl>
                    <FormDescription>Days until reward expires</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="inventory_total"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Inventory Total</FormLabel>
                    <FormControl>
                      <Input type="number" placeholder="1000" {...field} />
                    </FormControl>
                    <FormDescription>Leave empty for unlimited</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="config"
              render={({ field }) => (
                <FormItem>
                  <JsonEditor
                    value={field.value || '{}'}
                    onChange={field.onChange}
                    placeholder='{"key": "value"}'
                    rows={6}
                  />
                  <FormDescription>Additional configuration (JSON format)</FormDescription>
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
                {isEdit ? 'Update' : 'Create'} Reward
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
