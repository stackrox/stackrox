import React, { ReactElement } from 'react';
import Tippy, { TippyProps } from '@tippyjs/react';

type ExtendedTooltipProps = {
    type?: string;
};

export const defaultTippyTooltipProps = {
    arrow: true,
};

export type TooltipProps = TippyProps & ExtendedTooltipProps;

/**
 * Proxy component for `@tippyjs/react` that sets default behavior / props for
 * Tippy component. It's expected that this component will be used instead of
 * importing `@tippyjs/react` directly for the UI consistency.
 *
 * @see {@link TooltipOverlay} for a preferred content component to use for tooltip
 * @see {@link HoverHint} for adding tooltip to any DOM element in cases this component cannot be used
 */
function Tooltip(props: TooltipProps): ReactElement {
    const givenClasses = props.className || '';
    const extentedClassName =
        props.type === 'alert' ? `${givenClasses} alert-tooltip` : givenClasses;

    // eslint-disable-next-line react/jsx-props-no-spreading
    return <Tippy {...defaultTippyTooltipProps} {...props} className={extentedClassName} />;
}

export default Tooltip;
