import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { api } from '@/lib/api';
import { Reward } from '@/lib/types';
import { toast } from 'sonner';
import { Plus, Gift, Search, RefreshCw, MoreVertical, Upload, Edit } from 'lucide-react';
import { CreateRewardDialog } from '@/components/rewards/CreateRewardDialog';
import { UploadVoucherCodesDialog } from '@/components/rewards/UploadVoucherCodesDialog';
import { Empty } from '@/components/Empty';
import { formatCurrency, formatRewardType } from '@/lib/format';

export default function Rewards() {
  const [rewards, setRewards] = useState<Reward[]>([]);
  const [filteredRewards, setFilteredRewards] = useState<Reward[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingReward, setEditingReward] = useState<Reward | null>(null);
  const [uploadingCodesFor, setUploadingCodesFor] = useState<Reward | null>(null);

  useEffect(() => {
    loadRewards();
  }, []);

  useEffect(() => {
    filterRewards();
  }, [searchQuery, rewards]);

  const loadRewards = async () => {
    setIsLoading(true);
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

  const filterRewards = () => {
    if (!searchQuery.trim()) {
      setFilteredRewards(rewards);
      return;
    }

    const query = searchQuery.toLowerCase();
    const filtered = rewards.filter(
      (reward) =>
        reward.name.toLowerCase().includes(query) ||
        reward.type.toLowerCase().includes(query)
    );
    setFilteredRewards(filtered);
  };

  const handleToggleActive = async (reward: Reward) => {
    try {
      await api.rewards.update(reward.id, { active: !reward.active });
      toast.success(`Reward ${reward.active ? 'deactivated' : 'activated'}`);
      loadRewards();
    } catch (error: any) {
      toast.error('Failed to update reward status');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Rewards</h1>
          <p className="text-slate-600 mt-1">Manage your reward catalog</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadRewards}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Reward
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : rewards.length === 0 ? (
        <Empty
          icon={Gift}
          title="No rewards yet"
          description="Create your first reward to start your loyalty program"
          action={{
            label: 'Create Reward',
            onClick: () => setShowCreateDialog(true),
          }}
        />
      ) : (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Search rewards..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <span className="text-sm text-slate-600">
              {filteredRewards.length} of {rewards.length} rewards
            </span>
          </div>

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
                {filteredRewards.map((reward) => (
                  <TableRow key={reward.id}>
                    <TableCell className="font-medium">{reward.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{formatRewardType(reward.type)}</Badge>
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
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => setEditingReward(reward)}>
                            <Edit className="h-4 w-4 mr-2" />
                            Edit
                          </DropdownMenuItem>
                          {reward.type === 'voucher_code' && (
                            <DropdownMenuItem onClick={() => setUploadingCodesFor(reward)}>
                              <Upload className="h-4 w-4 mr-2" />
                              Upload Codes
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem onClick={() => handleToggleActive(reward)}>
                            {reward.active ? 'Deactivate' : 'Activate'}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      <CreateRewardDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={loadRewards}
      />

      <CreateRewardDialog
        open={!!editingReward}
        onOpenChange={(open) => !open && setEditingReward(null)}
        onSuccess={loadRewards}
        reward={editingReward}
      />

      <UploadVoucherCodesDialog
        reward={uploadingCodesFor}
        open={!!uploadingCodesFor}
        onOpenChange={(open) => !open && setUploadingCodesFor(null)}
        onSuccess={loadRewards}
      />
    </div>
  );
}
