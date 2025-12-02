import type { CSSProperties, ReactElement, ReactNode } from 'react';
import { Content, ContentVariants, Divider } from '@patternfly/react-core';

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
        '--pf-v5-u-max-height--MaxHeight': '75vh',
    } as CSSProperties;

    return (
        <div className={`pf-v6-u-max-height ${extraClassName}`} style={styleConstant}>
            <div>
                <Content
                    className="pf-v6-u-font-weight-bold"
                    component={ContentVariants.h3}
                    data-testid="tooltip-title"
                >
                    {title}
                </Content>
                {!!subtitle && <span data-testid="tooltip-subtitle">{subtitle}</span>}
            </div>
            <Divider />
            <div data-testid="tooltip-body">{body}</div>
            {!!footer && <footer data-testid="tooltip-footer">{footer}</footer>}
        </div>
    );
}

export default DetailedTooltipContent;
