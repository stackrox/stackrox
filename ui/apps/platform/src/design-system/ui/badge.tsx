import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from 'design-system/lib/utils';

const badgeVariants = cva(
    'inline-flex items-center rounded px-2 py-0.5 text-2xs font-600 uppercase tracking-wide transition-colors',
    {
        variants: {
            variant: {
                default: 'bg-bg-tertiary text-text-secondary border border-border-subtle',
                critical: 'bg-severity-critical/15 text-severity-critical',
                high: 'bg-severity-high/15 text-severity-high',
                medium: 'bg-severity-medium/15 text-severity-medium',
                low: 'bg-severity-low/15 text-severity-low',
                success: 'bg-success/15 text-success',
                info: 'bg-accent-blue/15 text-accent-blue',
            },
        },
        defaultVariants: { variant: 'default' },
    }
);

export interface BadgeProps
    extends React.HTMLAttributes<HTMLSpanElement>,
        VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
    return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
