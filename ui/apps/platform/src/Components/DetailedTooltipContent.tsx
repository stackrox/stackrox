import React, { ReactElement, ReactNode, CSSProperties } from 'react';
import { Divider, Text, TextVariants } from '@patternfly/react-core';

export type DetailedTooltipContentProps = {
    title: string;
    subtitle?: string;
    body: ReactNode;
    footer?: ReactNode;
    extraClassName?: string;
};

/**
 * Alternative to {@link TooltipOverlay} that provides layout for complex tooltip content with
 * title, subtitle and footer in addition to the main body.
 */
function DetailedTooltipContent({
    title,
    subtitle,
    body,
    footer,
    extraClassName = '',
}: DetailedTooltipContentProps): ReactElement | null {
    if (!title || !body) {
        return null;
    }

    const styleConstant = {
        overflow: 'scroll',
        '--pf-u-max-height--MaxHeight': '75vh',
    } as CSSProperties;

    return (
        <div className={`pf-u-max-height ${extraClassName}`} style={styleConstant}>
            <div>
                <Text
                    className="pf-u-font-weight-bold"
                    component={TextVariants.h3}
                    data-testid="tooltip-title"
                >
                    {title}
                </Text>
                {!!subtitle && <span data-testid="tooltip-subtitle">{subtitle}</span>}
            </div>
            <Divider />
            <div data-testid="tooltip-body">{body}</div>
            {!!footer && <footer data-testid="tooltip-footer">{footer}</footer>}
        </div>
    );
}

export default DetailedTooltipContent;
