import { cn } from '@/lib/utils';

export type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info' | 'primary';

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
}

const variants: Record<BadgeVariant, string> = {
  default:  'bg-neutral-100 text-neutral-700',
  primary:  'bg-primary-100 text-primary-700',
  success:  'bg-emerald-50 text-emerald-700',
  warning:  'bg-amber-50 text-amber-700',
  error:    'bg-red-50 text-red-700',
  info:     'bg-blue-50 text-blue-700',
};

export function Badge({ className, variant = 'default', children, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        variants[variant],
        className,
      )}
      {...props}
    >
      {children}
    </span>
  );
}
