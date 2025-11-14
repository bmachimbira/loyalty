import { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Plus, Trash2 } from 'lucide-react';
import { Alert, AlertDescription } from '../ui/alert';

interface Condition {
  field: string;
  operator: string;
  value: string;
}

interface RuleBuilderProps {
  eventType: string;
  onChange: (jsonLogic: any) => void;
  initialConditions?: Condition[];
}

export function RuleBuilder({ eventType, onChange, initialConditions }: RuleBuilderProps) {
  const [conditions, setConditions] = useState<Condition[]>(
    initialConditions || [{ field: '', operator: '==', value: '' }]
  );
  const [logicOperator, setLogicOperator] = useState<'all' | 'any'>('all');

  const eventFields: Record<string, string[]> = {
    purchase: ['amount', 'currency', 'category', 'merchant_id'],
    visit: ['location', 'duration', 'visit_type'],
    registration: ['source', 'channel'],
    referral: ['referee_id', 'status'],
    birthday: ['day', 'month'],
  };

  const operators = [
    { value: '==', label: 'equals' },
    { value: '!=', label: 'not equals' },
    { value: '>', label: 'greater than' },
    { value: '>=', label: 'greater than or equal' },
    { value: '<', label: 'less than' },
    { value: '<=', label: 'less than or equal' },
    { value: 'in', label: 'in list' },
  ];

  const buildJsonLogic = (conds: Condition[]) => {
    const rules = conds
      .filter((c) => c.field && c.value)
      .map((c) => {
        const value = isNaN(Number(c.value)) ? c.value : Number(c.value);
        if (c.operator === 'in') {
          return { in: [{ var: c.field }, value.toString().split(',').map((v) => v.trim())] };
        }
        return { [c.operator]: [{ var: c.field }, value] };
      });

    if (rules.length === 0) return {};
    if (rules.length === 1) return rules[0];
    return { [logicOperator]: rules };
  };

  const handleConditionChange = (index: number, field: keyof Condition, value: string) => {
    const newConditions = [...conditions];
    newConditions[index] = { ...newConditions[index], [field]: value };
    setConditions(newConditions);
    onChange(buildJsonLogic(newConditions));
  };

  const addCondition = () => {
    const newConditions = [...conditions, { field: '', operator: '==', value: '' }];
    setConditions(newConditions);
  };

  const removeCondition = (index: number) => {
    const newConditions = conditions.filter((_, i) => i !== index);
    setConditions(newConditions);
    onChange(buildJsonLogic(newConditions));
  };

  const handleLogicOperatorChange = (value: 'all' | 'any') => {
    setLogicOperator(value);
    onChange(buildJsonLogic(conditions));
  };

  const availableFields = eventFields[eventType] || ['amount', 'currency'];

  return (
    <div className="space-y-4">
      <Alert>
        <AlertDescription>
          Build your rule conditions. The rule will trigger when the conditions are met.
        </AlertDescription>
      </Alert>

      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">Match</span>
        <Select value={logicOperator} onValueChange={handleLogicOperatorChange}>
          <SelectTrigger className="w-[120px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All (AND)</SelectItem>
            <SelectItem value="any">Any (OR)</SelectItem>
          </SelectContent>
        </Select>
        <span className="text-sm text-slate-600">of the following conditions:</span>
      </div>

      <div className="space-y-3">
        {conditions.map((condition, index) => (
          <div key={index} className="flex items-center gap-2 bg-slate-50 p-3 rounded-lg">
            <Select
              value={condition.field}
              onValueChange={(value) => handleConditionChange(index, 'field', value)}
            >
              <SelectTrigger className="w-[150px]">
                <SelectValue placeholder="Field" />
              </SelectTrigger>
              <SelectContent>
                {availableFields.map((field) => (
                  <SelectItem key={field} value={field}>
                    {field}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={condition.operator}
              onValueChange={(value) => handleConditionChange(index, 'operator', value)}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {operators.map((op) => (
                  <SelectItem key={op.value} value={op.value}>
                    {op.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Input
              placeholder="Value"
              value={condition.value}
              onChange={(e) => handleConditionChange(index, 'value', e.target.value)}
              className="flex-1"
            />

            {conditions.length > 1 && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => removeCondition(index)}
              >
                <Trash2 className="h-4 w-4 text-red-500" />
              </Button>
            )}
          </div>
        ))}
      </div>

      <Button type="button" variant="outline" onClick={addCondition} className="w-full">
        <Plus className="h-4 w-4 mr-2" />
        Add Condition
      </Button>

      <div className="bg-slate-100 p-3 rounded-lg">
        <p className="text-xs text-slate-600 mb-1 font-medium">JSON Logic Preview:</p>
        <pre className="text-xs overflow-x-auto">
          {JSON.stringify(buildJsonLogic(conditions), null, 2)}
        </pre>
      </div>
    </div>
  );
}
