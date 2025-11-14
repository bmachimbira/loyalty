import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { createRuleSchema, CreateRuleInput } from '@/lib/schemas';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import { Rule, Reward } from '@/lib/types';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Loader2 } from 'lucide-react';
import { RuleBuilder } from './RuleBuilder';
import { JsonEditor } from '../JsonEditor';

interface CreateRuleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  rule?: Rule | null;
}

export function CreateRuleDialog({
  open,
  onOpenChange,
  onSuccess,
  rule,
}: CreateRuleDialogProps) {
  const isEdit = !!rule;
  const [rewards, setRewards] = useState<Reward[]>([]);
  const [builderMode, setBuilderMode] = useState(true);

  const form = useForm({
    resolver: zodResolver(createRuleSchema),
    defaultValues: {
      name: '',
      event_type: 'purchase' as const,
      conditions: {},
      reward_id: '',
      per_user_cap: 1,
      global_cap: '',
      cool_down_sec: 0,
      campaign_id: '',
      priority: 100,
    },
  });

  useEffect(() => {
    if (open) {
      loadRewards();
      if (rule) {
        form.reset({
          name: rule.name,
          event_type: rule.event_type,
          conditions: rule.conditions,
          reward_id: rule.reward_id,
          per_user_cap: rule.per_user_cap,
          global_cap: rule.global_cap?.toString() || '',
          cool_down_sec: rule.cool_down_sec,
          campaign_id: '',
          priority: 100,
        });
      }
    }
  }, [open, rule]);

  const loadRewards = async () => {
    try {
      const data = await api.rewards.list();
      setRewards(data.filter((r) => r.active));
    } catch (error) {
      console.error('Failed to load rewards');
    }
  };

  const onSubmit = async (data: CreateRuleInput) => {
    try {
      const payload = {
        name: data.name,
        event_type: data.event_type,
        conditions: data.conditions,
        reward_id: data.reward_id,
        per_user_cap: Number(data.per_user_cap),
        global_cap: data.global_cap ? Number(data.global_cap) : undefined,
        cool_down_sec: Number(data.cool_down_sec),
      };

      if (isEdit && rule) {
        await api.rules.update(rule.id, payload);
        toast.success('Rule updated successfully');
      } else {
        await api.rules.create(payload);
        toast.success('Rule created successfully');
      }

      form.reset();
      onOpenChange(false);
      onSuccess();
    } catch (error: any) {
      toast.error(error.message || `Failed to ${isEdit ? 'update' : 'create'} rule`);
    }
  };

  const eventType = form.watch('event_type');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Rule' : 'Create Rule'}</DialogTitle>
          <DialogDescription>
            {isEdit ? 'Update rule details' : 'Define a new rule to automatically issue rewards'}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Rule Name *</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., Spend $50 Get $5 Discount" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="event_type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Event Type *</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="purchase">Purchase</SelectItem>
                        <SelectItem value="visit">Visit</SelectItem>
                        <SelectItem value="registration">Registration</SelectItem>
                        <SelectItem value="referral">Referral</SelectItem>
                        <SelectItem value="birthday">Birthday</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="reward_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Reward *</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select reward" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {rewards.map((reward) => (
                          <SelectItem key={reward.id} value={reward.id}>
                            {reward.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="conditions"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Rule Conditions *</FormLabel>
                  <Tabs value={builderMode ? 'builder' : 'json'} onValueChange={(v) => setBuilderMode(v === 'builder')}>
                    <TabsList className="grid w-full grid-cols-2">
                      <TabsTrigger value="builder">Visual Builder</TabsTrigger>
                      <TabsTrigger value="json">JSON Editor</TabsTrigger>
                    </TabsList>
                    <TabsContent value="builder" className="mt-4">
                      <RuleBuilder
                        eventType={eventType}
                        onChange={(jsonLogic) => field.onChange(jsonLogic)}
                      />
                    </TabsContent>
                    <TabsContent value="json" className="mt-4">
                      <JsonEditor
                        value={JSON.stringify(field.value || {}, null, 2)}
                        onChange={(value) => {
                          try {
                            field.onChange(JSON.parse(value));
                          } catch (e) {
                            // Invalid JSON, ignore
                          }
                        }}
                        rows={8}
                      />
                    </TabsContent>
                  </Tabs>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="grid grid-cols-3 gap-4">
              <FormField
                control={form.control}
                name="per_user_cap"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Per User Cap *</FormLabel>
                    <FormControl>
                      <Input type="number" min="1" {...field} />
                    </FormControl>
                    <FormDescription>Max times per user</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="global_cap"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Global Cap</FormLabel>
                    <FormControl>
                      <Input type="number" min="1" placeholder="Unlimited" {...field} />
                    </FormControl>
                    <FormDescription>Max total issuances</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="cool_down_sec"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Cooldown (sec)</FormLabel>
                    <FormControl>
                      <Input type="number" min="0" {...field} />
                    </FormControl>
                    <FormDescription>Min time between</FormDescription>
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
                {isEdit ? 'Update' : 'Create'} Rule
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
