import React, { ReactElement } from 'react';
import Tippy, { TippyProps } from '@tippyjs/react';

export const defaultTippyTooltipProps = {
    arrow: true,
};

/**
 * Proxy component for `@tippyjs/react` that sets default behavior / props for
 * Tippy component. It's expected that this component will be used instead of
 * importing `@tippyjs/react` directly for the UI consistency.
 *
 * @see {@link TooltipOverlay} for a preferred content component to use for tooltip
 * @see {@link HoverHint} for adding tooltip to any DOM element in cases this component cannot be used
 */
function Tooltip(props: TippyProps): ReactElement {
    // eslint-disable-next-line react/jsx-props-no-spreading
    return <Tippy {...defaultTippyTooltipProps} {...props} />;
}

export type TooltipProps = TippyProps;
export default Tooltip;
