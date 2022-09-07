import React, { ReactElement, ReactNode } from 'react';

import { Popover } from '@patternfly/react-core';

export type ToggledPopoverElementProps = {
    children: ReactNode;
    popoverContent: string;
    ariaLabel: string;
    className: string;
};

/*
 * Display children in a message box
 * that is centered in the full height and width of its parent.
 */
const ToggledPopoverElement = ({
    children,
    popoverContent,
    ariaLabel,
    className,
}: ToggledPopoverElementProps): ReactElement => (
    <Popover showClose={false} bodyContent={popoverContent}>
        <button
            type="button"
            aria-label={ariaLabel}
            onClick={(e) => e.preventDefault()}
            aria-describedby="simple-form-name-01"
            className={className}
        >
            {children}
        </button>
    </Popover>
);

export default ToggledPopoverElement;
