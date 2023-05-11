import React from 'react';
import { Tooltip } from '@patternfly/react-core';
import { Th } from '@patternfly/react-table';

type TooltipThProps = {
    children: string | React.ReactNode;
    tooltip: string;
};

// this is to ensure that the tooltip always shows up on hover since
// the tooltip prop on Th only shows when the header is truncated
function TooltipTh({ children, tooltip }: TooltipThProps) {
    return (
        <Th>
            <Tooltip content={tooltip} position="top-start" isContentLeftAligned>
                <div>{children}</div>
            </Tooltip>
        </Th>
    );
}

export default TooltipTh;
