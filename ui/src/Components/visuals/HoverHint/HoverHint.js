import { useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
import PropTypes from 'prop-types';
import tippy from 'tippy.js';

/**
 * This component is supposed to be used only in case a hover hint / tooltip
 * needs to be created for a DOM element directly. That usually comes in handy
 * when some external library controls rendering and we can only react to mouse
 * events that return DOM element as a target (e.g. charting library returns
 * SVG element).
 *
 * In case element is being rendered directly, consider using `Tippy` component
 * from `@tippy.js/react` instead of this one.
 *
 * This component proxies props (besides `target` and `children`) to the created
 * instance of `tippy.js`.
 */
const HoverHint = ({ target, children, ...props }) => {
    const elRef = useRef(null);
    // to avoid creating an element on every render
    if (!elRef.current) {
        elRef.current = document.createElement('div');
    }

    useEffect(
        () => {
            document.body.appendChild(elRef.current);
            const tippyInstance = tippy(target, {
                content: elRef.current,
                ...props
            });
            tippyInstance.show();

            return () => {
                if (typeof tippyInstance.destroy === 'function') {
                    tippyInstance.destroy();
                }
            };
        },
        [props, target]
    );

    return createPortal(children, elRef.current);
};

HoverHint.propTypes = {
    /** target DOM element */
    target: PropTypes.instanceOf(Element).isRequired,
    /** content to render when hint appears */
    children: PropTypes.node.isRequired
};

export default HoverHint;
