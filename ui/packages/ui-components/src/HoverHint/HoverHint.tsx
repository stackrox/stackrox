import { useRef, useEffect, ReactElement } from 'react';
import { createPortal } from 'react-dom';
import PropTypes, { InferProps } from 'prop-types';
import tippy from 'tippy.js';

import { defaultTippyTooltipProps } from '../Tooltip';

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
 * @see {@link Tooltip} for defining tooltips directly in JSX
 */
function HoverHint({ target, children, ...props }: HoverHintProps): ReactElement {
    const elRef = useRef<Element>();
    // to avoid creating an element on every render
    if (!elRef.current) {
        elRef.current = document.createElement('div');
    }

    useEffect(() => {
        if (!elRef.current) {
            return undefined;
        }

        document.body.appendChild(elRef.current);
        const tippyInstance = tippy(target, {
            content: elRef.current,
            ...defaultTippyTooltipProps,
            ...props,
        });
        tippyInstance.show();

        return (): void => {
            if (typeof tippyInstance.destroy === 'function') {
                tippyInstance.destroy();
            }
        };
    }, [props, target]);

    return createPortal(children, elRef.current);
}

HoverHint.propTypes = {
    /** target DOM element */
    target: PropTypes.instanceOf(Element).isRequired,
    /** content to render when hint appears */
    children: PropTypes.node.isRequired,
};

export type HoverHintProps = InferProps<typeof HoverHint.propTypes>;
export default HoverHint;
