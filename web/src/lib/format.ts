// Utility functions for formatting data

export function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function formatDateTime(dateString: string): string {
  return new Date(dateString).toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatCurrency(
  amount: number | null,
  currency: string | null = 'ZWG'
): string {
  if (amount === null) return '-';
  return `${currency || 'ZWG'} ${amount.toFixed(2)}`;
}

export function formatNumber(value: number | null): string {
  if (value === null) return '-';
  return value.toLocaleString('en-US');
}

export function formatPercentage(value: number | null, decimals: number = 1): string {
  if (value === null) return '-';
  return `${(value * 100).toFixed(decimals)}%`;
}

export function formatPhone(phone: string | null): string {
  if (!phone) return '-';
  return phone;
}

export function formatRewardType(type: string): string {
  const typeMap: Record<string, string> = {
    discount: 'Discount',
    voucher_code: 'Voucher Code',
    points_credit: 'Points Credit',
    external_voucher: 'External Voucher',
    physical_item: 'Physical Item',
    webhook_custom: 'Webhook Custom',
  };
  return typeMap[type] || type;
}

export function formatEventType(type: string): string {
  return type
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

export function parseJSON(value: string): any {
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

export function stringifyJSON(value: any): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return '';
  }
}
