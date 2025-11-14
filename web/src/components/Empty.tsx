import { LucideIcon } from 'lucide-react';
import { Button } from './ui/button';

interface EmptyProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export function Empty({ icon: Icon, title, description, action }: EmptyProps) {
  return (
    <div className="text-center py-12 bg-white rounded-lg border border-slate-200">
      <Icon className="h-12 w-12 text-slate-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-slate-900 mb-2">{title}</h3>
      <p className="text-slate-600 mb-4">{description}</p>
      {action && (
        <Button onClick={action.onClick}>{action.label}</Button>
      )}
    </div>
  );
}
