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
import { api } from '@/lib/api';
import { Campaign } from '@/lib/types';
import { toast } from 'sonner';
import { Plus, Megaphone, Search, RefreshCw, Edit } from 'lucide-react';
import { CreateCampaignDialog } from '@/components/campaigns/CreateCampaignDialog';
import { Empty } from '@/components/Empty';
import { formatDate } from '@/lib/format';

export default function Campaigns() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [filteredCampaigns, setFilteredCampaigns] = useState<Campaign[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingCampaign, setEditingCampaign] = useState<Campaign | null>(null);

  useEffect(() => {
    loadCampaigns();
  }, []);

  useEffect(() => {
    filterCampaigns();
  }, [searchQuery, campaigns]);

  const loadCampaigns = async () => {
    setIsLoading(true);
    try {
      const data = await api.campaigns.list();
      setCampaigns(data);
    } catch (error: any) {
      toast.error('Failed to load campaigns');
      console.error(error);
    } finally {
      setIsLoading(false);
    }
  };

  const filterCampaigns = () => {
    if (!searchQuery.trim()) {
      setFilteredCampaigns(campaigns);
      return;
    }

    const query = searchQuery.toLowerCase();
    const filtered = campaigns.filter((campaign) =>
      campaign.name.toLowerCase().includes(query)
    );
    setFilteredCampaigns(filtered);
  };

  const getCampaignStatus = (campaign: Campaign) => {
    const now = new Date();
    const start = new Date(campaign.start_date);
    const end = new Date(campaign.end_date);

    if (now < start) return 'planned';
    if (now > end) return 'ended';
    return 'active';
  };

  const getStatusVariant = (status: string) => {
    switch (status) {
      case 'active':
        return 'default';
      case 'planned':
        return 'secondary';
      case 'ended':
        return 'outline';
      default:
        return 'outline';
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Campaigns</h1>
          <p className="text-slate-600 mt-1">Manage your marketing campaigns</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadCampaigns}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Campaign
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : campaigns.length === 0 ? (
        <Empty
          icon={Megaphone}
          title="No campaigns yet"
          description="Create campaigns to organize your loyalty initiatives"
          action={{
            label: 'Create Campaign',
            onClick: () => setShowCreateDialog(true),
          }}
        />
      ) : (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Search campaigns..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <span className="text-sm text-slate-600">
              {filteredCampaigns.length} of {campaigns.length} campaigns
            </span>
          </div>

          <div className="bg-white rounded-lg border border-slate-200">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Start Date</TableHead>
                  <TableHead>End Date</TableHead>
                  <TableHead>Budget</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredCampaigns.map((campaign) => {
                  const status = getCampaignStatus(campaign);
                  return (
                    <TableRow key={campaign.id}>
                      <TableCell className="font-medium">{campaign.name}</TableCell>
                      <TableCell>{formatDate(campaign.start_date)}</TableCell>
                      <TableCell>{formatDate(campaign.end_date)}</TableCell>
                      <TableCell>{campaign.budget?.name || 'No budget'}</TableCell>
                      <TableCell>
                        <Badge variant={getStatusVariant(status)}>
                          {status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setEditingCampaign(campaign)}
                        >
                          <Edit className="h-4 w-4 mr-2" />
                          Edit
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      <CreateCampaignDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={loadCampaigns}
      />

      <CreateCampaignDialog
        open={!!editingCampaign}
        onOpenChange={(open) => !open && setEditingCampaign(null)}
        onSuccess={loadCampaigns}
        campaign={editingCampaign}
      />
    </div>
  );
}
