import { useState } from 'react';
import { api } from '@/lib/api';
import { toast } from 'sonner';
import { Reward } from '@/lib/types';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2, Upload, AlertCircle } from 'lucide-react';

interface UploadVoucherCodesDialogProps {
  reward: Reward | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function UploadVoucherCodesDialog({
  reward,
  open,
  onOpenChange,
  onSuccess,
}: UploadVoucherCodesDialogProps) {
  const [codes, setCodes] = useState('');
  const [isUploading, setIsUploading] = useState(false);

  const handleUpload = async () => {
    if (!reward) return;

    const codeList = codes
      .split('\n')
      .map((code) => code.trim())
      .filter((code) => code.length > 0);

    if (codeList.length === 0) {
      toast.error('Please enter at least one voucher code');
      return;
    }

    setIsUploading(true);
    try {
      const result = await api.rewards.uploadCodes(reward.id, codeList);
      toast.success(`Successfully uploaded ${result.uploaded} voucher codes`);
      setCodes('');
      onOpenChange(false);
      onSuccess();
    } catch (error: any) {
      toast.error(error.message || 'Failed to upload voucher codes');
    } finally {
      setIsUploading(false);
    }
  };

  const codeCount = codes
    .split('\n')
    .map((code) => code.trim())
    .filter((code) => code.length > 0).length;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Upload Voucher Codes</DialogTitle>
          <DialogDescription>
            Upload codes for: <strong>{reward?.name}</strong>
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Enter one voucher code per line. Codes will be assigned when this reward is
              issued to customers.
            </AlertDescription>
          </Alert>

          <div>
            <label className="text-sm font-medium mb-2 block">Voucher Codes</label>
            <Textarea
              value={codes}
              onChange={(e) => setCodes(e.target.value)}
              placeholder="CODE1&#10;CODE2&#10;CODE3"
              rows={10}
              className="font-mono text-sm"
            />
            <p className="text-sm text-slate-600 mt-2">
              {codeCount} code{codeCount !== 1 ? 's' : ''} ready to upload
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isUploading}
          >
            Cancel
          </Button>
          <Button onClick={handleUpload} disabled={isUploading || codeCount === 0}>
            {isUploading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            <Upload className="mr-2 h-4 w-4" />
            Upload Codes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
