import React, { ReactElement } from 'react';
import PropTypes, { InferProps } from 'prop-types';

/**
 * Tooltip overlay container that provides default styling for any tooltip content.
 *
 * @see {@link Tooltip}
 */
function TooltipOverlay({ extraClassName, children }: TooltipOverlayProps): ReactElement {
    return <div className={`rox-tooltip-overlay p-2 ${extraClassName}`}>{children}</div>;
}

TooltipOverlay.propTypes = {
    /** CSS classes to add to the top level HTML element */
    extraClassName: PropTypes.string,
    /** tooltip content */
    children: PropTypes.node.isRequired,
};

TooltipOverlay.defaultProps = {
    extraClassName: '',
};

export type TooltipOverlayProps = InferProps<typeof TooltipOverlay.propTypes>;
export default TooltipOverlay;
