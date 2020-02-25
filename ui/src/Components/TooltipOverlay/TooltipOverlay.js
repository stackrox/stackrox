import React, { useRef } from 'react';
import PropTypes from 'prop-types';

import { adjustTooltipPosition } from 'utils/domUtils';

const TooltipOverlay = ({ top, left, className, title, subtitle, body, footer }) => {
    const tooltipRef = useRef(null);
    if (!title || !body) return null;

    const [adjustedTop, adjustedLeft] = adjustTooltipPosition(top, left, tooltipRef);

    return (
        <div
            className={`graph-hint text-xs text-base-600 z-10 bg-tertiary-200 rounded min-w-32 border border-tertiary-400 ${className}`}
            style={{ top: adjustedTop, left: adjustedLeft }}
            ref={tooltipRef}
        >
            <div className="flex flex-col border-b border-primary-400 mb-1 py-1 px-2 leading-loose">
                <h2 className="graph-hint-title text-sm">{title}</h2>
                {subtitle && <span>{subtitle}</span>}
            </div>
            <div className="graph-hint-body px-2 pt-1 pb-2">{body}</div>
            {!!footer && (
                <footer className="font-700 text-sm leading-loose px-2 pb-1">{footer}</footer>
            )}
        </div>
    );
};

TooltipOverlay.propTypes = {
    top: PropTypes.number,
    left: PropTypes.number,
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired,
    subtitle: PropTypes.string,
    footer: PropTypes.node,
    className: PropTypes.string
};

TooltipOverlay.defaultProps = {
    top: 0,
    left: 0,
    subtitle: '',
    footer: '',
    className: ''
};

export default TooltipOverlay;
