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
import { Rule } from '@/lib/types';
import { toast } from 'sonner';
import { Plus, Zap } from 'lucide-react';

export default function Rules() {
  const [rules, setRules] = useState<Rule[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadRules();
  }, []);

  const loadRules = async () => {
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

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-slate-900">Rules</h1>
          <p className="text-slate-600 mt-1">Define when and how rewards are issued</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create Rule
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : rules.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-slate-200">
          <Zap className="h-12 w-12 text-slate-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-slate-900 mb-2">No rules yet</h3>
          <p className="text-slate-600 mb-4">Create rules to automatically issue rewards based on customer events</p>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Create Rule
          </Button>
        </div>
      ) : (
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
              {rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell className="font-medium">{rule.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{rule.event_type}</Badge>
                  </TableCell>
                  <TableCell>{rule.reward?.name || '-'}</TableCell>
                  <TableCell>{rule.per_user_cap}</TableCell>
                  <TableCell>
                    <Badge variant={rule.active ? 'default' : 'secondary'}>
                      {rule.active ? 'Active' : 'Inactive'}
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
