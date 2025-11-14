import { useEffect, useState } from 'react';
import { Customer, Issuance } from '@/lib/types';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { formatDate, formatCurrency } from '@/lib/format';
import { Loader2 } from 'lucide-react';

interface CustomerDetailDialogProps {
  customer: Customer | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdate: () => void;
}

export function CustomerDetailDialog({
  customer,
  open,
  onOpenChange,
  onUpdate,
}: CustomerDetailDialogProps) {
  const [issuances, setIssuances] = useState<Issuance[]>([]);
  const [loading, setLoading] = useState(false);
  const [updatingStatus, setUpdatingStatus] = useState(false);

  useEffect(() => {
    if (customer && open) {
      loadIssuances();
    }
  }, [customer, open]);

  const loadIssuances = async () => {
    if (!customer) return;
    setLoading(true);
    try {
      const data = await api.issuances.list(customer.id);
      setIssuances(data);
    } catch (error: any) {
      toast.error('Failed to load issuances');
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = async (newStatus: 'active' | 'suspended' | 'deleted') => {
    if (!customer) return;
    setUpdatingStatus(true);
    try {
      await api.customers.updateStatus(customer.id, newStatus);
      toast.success('Customer status updated');
      onUpdate();
    } catch (error: any) {
      toast.error('Failed to update status');
    } finally {
      setUpdatingStatus(false);
    }
  };

  const getStatusVariant = (status: string) => {
    switch (status) {
      case 'reserved':
      case 'issued':
        return 'default';
      case 'redeemed':
        return 'secondary';
      case 'expired':
      case 'cancelled':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  if (!customer) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Customer Details</DialogTitle>
          <DialogDescription>
            View and manage customer information
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium text-slate-600">Phone</label>
              <p className="text-sm">{customer.phone_e164 || '-'}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-slate-600">External Ref</label>
              <p className="text-sm">{customer.external_ref || '-'}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-slate-600">Created</label>
              <p className="text-sm">{formatDate(customer.created_at)}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-slate-600">Status</label>
              <div className="flex gap-2 items-center">
                <Select
                  value={customer.status}
                  onValueChange={handleStatusChange}
                  disabled={updatingStatus}
                >
                  <SelectTrigger className="w-[140px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="suspended">Suspended</SelectItem>
                    <SelectItem value="deleted">Deleted</SelectItem>
                  </SelectContent>
                </Select>
                {updatingStatus && <Loader2 className="h-4 w-4 animate-spin" />}
              </div>
            </div>
          </div>

          <Tabs defaultValue="issuances" className="w-full">
            <TabsList>
              <TabsTrigger value="issuances">Rewards ({issuances.length})</TabsTrigger>
            </TabsList>
            <TabsContent value="issuances" className="mt-4">
              {loading ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin" />
                </div>
              ) : issuances.length === 0 ? (
                <p className="text-center text-slate-600 py-8">No rewards issued yet</p>
              ) : (
                <div className="border rounded-lg">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Reward</TableHead>
                        <TableHead>Value</TableHead>
                        <TableHead>State</TableHead>
                        <TableHead>Issued</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {issuances.map((issuance) => (
                        <TableRow key={issuance.id}>
                          <TableCell className="font-medium">
                            {issuance.reward?.name || '-'}
                          </TableCell>
                          <TableCell>
                            {formatCurrency(issuance.face_value, issuance.currency)}
                          </TableCell>
                          <TableCell>
                            <Badge variant={getStatusVariant(issuance.state)}>
                              {issuance.state}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            {issuance.issued_at ? formatDate(issuance.issued_at) : '-'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </TabsContent>
          </Tabs>
        </div>
      </DialogContent>
    </Dialog>
  );
}
