import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from 'design-system/lib/utils';

const buttonVariants = cva(
    'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded text-sm font-500 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-blue focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
    {
        variants: {
            variant: {
                default: 'bg-accent-blue text-white hover:bg-accent-blue/90',
                destructive:
                    'bg-severity-critical/15 text-severity-critical border border-severity-critical/30 hover:bg-severity-critical/25',
                outline:
                    'border border-border bg-bg-secondary text-text-secondary hover:bg-bg-hover hover:text-text-primary',
                secondary:
                    'bg-bg-tertiary text-text-secondary border border-border hover:bg-bg-hover',
                ghost: 'hover:bg-bg-hover text-text-secondary hover:text-text-primary',
                link: 'text-accent-blue underline-offset-4 hover:underline',
            },
            size: {
                default: 'h-9 px-4 py-2',
                sm: 'h-7 rounded px-3 text-xs',
                lg: 'h-10 rounded px-8',
                icon: 'h-9 w-9',
            },
        },
        defaultVariants: { variant: 'default', size: 'default' },
    }
);

export interface ButtonProps
    extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
    asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    ({ className, variant, size, asChild = false, ...props }, ref) => {
        const Comp = asChild ? Slot : 'button';
        return (
            <Comp
                className={cn(buttonVariants({ variant, size, className }))}
                ref={ref}
                {...props}
            />
        );
    }
);
Button.displayName = 'Button';

export { Button, buttonVariants };
