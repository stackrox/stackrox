import type { ReactElement, ReactNode } from 'react';
import { Tooltip } from '@patternfly/react-core';
import { Th } from '@patternfly/react-table';
import type { ThProps } from '@patternfly/react-table';

type TooltipThProps = {
    className?: string;
    children: string | ReactNode;
    sort?: ThProps['sort'];
    tooltip: string;
};

// this is to ensure that the tooltip always shows up on hover since
// the tooltip prop on Th only shows when the header is truncated
function TooltipTh({ className, children, tooltip, sort }: TooltipThProps): ReactElement {
    return (
        <Th className={className} sort={sort || undefined}>
            <Tooltip content={tooltip} position="top-start" isContentLeftAligned>
                <div>{children}</div>
            </Tooltip>
        </Th>
    );
}

export default TooltipTh;
