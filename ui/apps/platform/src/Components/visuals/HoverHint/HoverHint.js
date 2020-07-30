import { useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import PropTypes from 'prop-types';
// this component targets DOM elements, and therefore compliments Tooltip component
// eslint-disable-next-line no-restricted-imports
import tippy from 'tippy.js';

import { defaultTippyTooltipProps } from 'Components/Tooltip';

/**
 * This component is supposed to be used only in case a hover hint / tooltip
 * needs to be created for a DOM element directly. That usually comes in handy
 * when some external library controls rendering and we can only react to mouse
 * events that return DOM element as a target (e.g. charting library returns
 * SVG element).
 *
 * This component proxies props (besides `target` and `children`) to the
 * instance of `tippy.js`.
 *
 * @see {@link Components/Tooltip} for defining tooltips directly in JSX
 */
const HoverHint = ({ target, children, ...props }) => {
    const elRef = useRef(null);
    // to avoid creating an element on every render
    if (!elRef.current) {
        elRef.current = document.createElement('div');
    }

    useEffect(() => {
        document.body.appendChild(elRef.current);
        const tippyInstance = tippy(target, {
            content: elRef.current,
            ...defaultTippyTooltipProps,
            ...props,
        });
        tippyInstance.show();

        return () => {
            if (typeof tippyInstance.destroy === 'function') {
                tippyInstance.destroy();
            }
        };
    }, [props, target]);

    return createPortal(children, elRef.current);
};

HoverHint.propTypes = {
    /** target DOM element */
    target: PropTypes.instanceOf(Element).isRequired,
    /** content to render when hint appears */
    children: PropTypes.node.isRequired,
};

export default HoverHint;
