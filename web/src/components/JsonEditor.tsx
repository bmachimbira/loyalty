import { useState } from 'react';
import { Textarea } from './ui/textarea';
import { Alert, AlertDescription } from './ui/alert';
import { AlertCircle } from 'lucide-react';

interface JsonEditorProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rows?: number;
}

export function JsonEditor({ value, onChange, placeholder, rows = 10 }: JsonEditorProps) {
  const [error, setError] = useState<string | null>(null);

  const handleChange = (newValue: string) => {
    onChange(newValue);

    // Validate JSON if not empty
    if (newValue.trim()) {
      try {
        JSON.parse(newValue);
        setError(null);
      } catch (e: any) {
        setError(e.message);
      }
    } else {
      setError(null);
    }
  };

  const formatJSON = () => {
    try {
      const parsed = JSON.parse(value);
      const formatted = JSON.stringify(parsed, null, 2);
      onChange(formatted);
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex justify-between items-center">
        <label className="text-sm font-medium">JSON Data</label>
        <button
          type="button"
          onClick={formatJSON}
          className="text-sm text-blue-600 hover:text-blue-800"
        >
          Format JSON
        </button>
      </div>
      <Textarea
        value={value}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={placeholder || 'Enter JSON data...'}
        rows={rows}
        className="font-mono text-sm"
      />
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>Invalid JSON: {error}</AlertDescription>
        </Alert>
      )}
    </div>
  );
}
