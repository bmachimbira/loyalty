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
import { Rule } from '@/lib/types';
import { toast } from 'sonner';
import { Plus, Zap, Search, RefreshCw, MoreVertical, Edit, Trash2 } from 'lucide-react';
import { CreateRuleDialog } from '@/components/rules/CreateRuleDialog';
import { Empty } from '@/components/Empty';
import { formatEventType } from '@/lib/format';

export default function Rules() {
  const [rules, setRules] = useState<Rule[]>([]);
  const [filteredRules, setFilteredRules] = useState<Rule[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingRule, setEditingRule] = useState<Rule | null>(null);

  useEffect(() => {
    loadRules();
  }, []);

  useEffect(() => {
    filterRules();
  }, [searchQuery, rules]);

  const loadRules = async () => {
    setIsLoading(true);
    try {
      const data = await api.rules.list();
      setRules(data);
    } catch (error: any) {
      toast.error('Failed to load rules');
      console.error(error);
    } finally {
      setIsLoading(false);
    }
  };

  const filterRules = () => {
    if (!searchQuery.trim()) {
      setFilteredRules(rules);
      return;
    }

    const query = searchQuery.toLowerCase();
    const filtered = rules.filter(
      (rule) =>
        rule.name.toLowerCase().includes(query) ||
        rule.event_type.toLowerCase().includes(query)
    );
    setFilteredRules(filtered);
  };

  const handleToggleActive = async (rule: Rule) => {
    try {
      await api.rules.update(rule.id, { active: !rule.active });
      toast.success(`Rule ${rule.active ? 'deactivated' : 'activated'}`);
      loadRules();
    } catch (error: any) {
      toast.error('Failed to update rule status');
    }
  };

  const handleDelete = async (rule: Rule) => {
    if (!confirm(`Are you sure you want to delete the rule "${rule.name}"?`)) {
      return;
    }
    try {
      await api.rules.delete(rule.id);
      toast.success('Rule deleted successfully');
      loadRules();
    } catch (error: any) {
      toast.error('Failed to delete rule');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Rules</h1>
          <p className="text-slate-600 mt-1">Define when and how rewards are issued</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadRules}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Rule
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : rules.length === 0 ? (
        <Empty
          icon={Zap}
          title="No rules yet"
          description="Create rules to automatically issue rewards based on customer events"
          action={{
            label: 'Create Rule',
            onClick: () => setShowCreateDialog(true),
          }}
        />
      ) : (
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Search rules..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <span className="text-sm text-slate-600">
              {filteredRules.length} of {rules.length} rules
            </span>
          </div>

          <div className="bg-white rounded-lg border border-slate-200">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Event Type</TableHead>
                  <TableHead>Reward</TableHead>
                  <TableHead>Per-User Cap</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredRules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className="font-medium">{rule.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{formatEventType(rule.event_type)}</Badge>
                    </TableCell>
                    <TableCell>{rule.reward?.name || '-'}</TableCell>
                    <TableCell>{rule.per_user_cap}</TableCell>
                    <TableCell>
                      <Badge variant={rule.active ? 'default' : 'secondary'}>
                        {rule.active ? 'Active' : 'Inactive'}
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
                          <DropdownMenuItem onClick={() => setEditingRule(rule)}>
                            <Edit className="h-4 w-4 mr-2" />
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleToggleActive(rule)}>
                            {rule.active ? 'Deactivate' : 'Activate'}
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleDelete(rule)}
                            className="text-red-600"
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
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

      <CreateRuleDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSuccess={loadRules}
      />

      <CreateRuleDialog
        open={!!editingRule}
        onOpenChange={(open) => !open && setEditingRule(null)}
        onSuccess={loadRules}
        rule={editingRule}
      />
    </div>
  );
}
