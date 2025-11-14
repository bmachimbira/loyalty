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
import { Reward } from '@/lib/types';
import { toast } from 'sonner';
import { Plus, Gift } from 'lucide-react';

export default function Rewards() {
  const [rewards, setRewards] = useState<Reward[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadRewards();
  }, []);

  const loadRewards = async () => {
    try {
      const data = await api.rewards.list();
      setRewards(data);
    } catch (error: any) {
      toast.error('Failed to load rewards');
      console.error(error);
    } finally {
      setIsLoading(false);
    }
  };

  const formatCurrency = (amount: number | null, currency: string | null) => {
    if (amount === null) return '-';
    return `${currency || 'ZWG'} ${amount.toFixed(2)}`;
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Rewards</h1>
          <p className="text-slate-600 mt-1">Manage your reward catalog</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create Reward
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : rewards.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-slate-200">
          <Gift className="h-12 w-12 text-slate-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-slate-900 mb-2">No rewards yet</h3>
          <p className="text-slate-600 mb-4">Create your first reward to start your loyalty program</p>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Create Reward
          </Button>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-slate-200">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Face Value</TableHead>
                <TableHead>Inventory</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rewards.map((reward) => (
                <TableRow key={reward.id}>
                  <TableCell className="font-medium">{reward.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{reward.type}</Badge>
                  </TableCell>
                  <TableCell>
                    {formatCurrency(reward.face_value, reward.currency)}
                  </TableCell>
                  <TableCell>
                    {reward.inventory_total
                      ? `${reward.inventory_used}/${reward.inventory_total}`
                      : 'Unlimited'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={reward.active ? 'default' : 'secondary'}>
                      {reward.active ? 'Active' : 'Inactive'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="sm">
                      Edit
                    </Button>
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
