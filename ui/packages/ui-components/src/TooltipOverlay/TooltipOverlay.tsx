import React, { ReactElement, ReactNode } from 'react';

export type TooltipOverlayProps = {
    /** CSS classes to add to the top level HTML element */
    extraClassName?: string;
    /** tooltip content */
    children: ReactNode;
};

/**
 * Tooltip overlay container that provides default styling for any tooltip content.
 *
 * @see {@link Tooltip}
 */
function TooltipOverlay({ extraClassName = '', children }: TooltipOverlayProps): ReactElement {
    return <div className={`rox-tooltip-overlay p-2 ${extraClassName}`}>{children}</div>;
}

export default TooltipOverlay;
