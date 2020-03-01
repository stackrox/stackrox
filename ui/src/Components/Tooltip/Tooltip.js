import React from 'react';

// That's the Tooltip component that proxies Tippy
// eslint-disable-next-line no-restricted-imports
import Tippy from '@tippy.js/react';

export const defaultTippyTooltipProps = {
    arrow: true
};

/**
 * Proxy component for `@tippy.js/react` that sets default behavior / props for
 * Tippy component. It's expected that this component will be used instead of
 * importing `@tippy.js/react` directly for the UI consistency.
 *
 * @see {@link Components/TooltipOverlay} for a preferred content component to use for tooltip
 * @see {@link Components/HoverHint} for adding tooltip to any DOM element in cases this component cannot be used
 */
const Tooltip = props => <Tippy {...defaultTippyTooltipProps} {...props} />;

export default Tooltip;
