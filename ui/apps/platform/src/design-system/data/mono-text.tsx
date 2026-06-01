import { cn } from 'design-system/lib/utils';

interface MonoTextProps extends React.HTMLAttributes<HTMLSpanElement> {
    children: React.ReactNode;
}

export function MonoText({ className, children, ...props }: MonoTextProps) {
    return <span className={cn('font-mono text-xs', className)} {...props}>{children}</span>;
}
