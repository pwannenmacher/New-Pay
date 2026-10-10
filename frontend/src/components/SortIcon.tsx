import { IconChevronDown, IconChevronUp } from '@tabler/icons-react';

interface SortIconProps {
  column: string;
  sortBy: string;
  sortOrder: 'asc' | 'desc';
}

export function SortIcon({ column, sortBy, sortOrder }: SortIconProps) {
  if (sortBy !== column) return null;
  return sortOrder === 'asc' ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />;
}
