import { cn } from 'design-system/lib/utils';

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
    return <div className={cn('animate-pulse rounded bg-bg-tertiary', className)} {...props} />;
}

export { Skeleton };
