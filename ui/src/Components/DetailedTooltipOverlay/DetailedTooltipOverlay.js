import React from 'react';
import PropTypes from 'prop-types';

import TooltipOverlay from 'Components/TooltipOverlay';

const DetailedTooltipOverlay = ({ className, title, subtitle, body, footer }) => {
    if (!title || !body) return null;

    return (
        <TooltipOverlay className={`min-w-32 ${className}`}>
            <div className="text-left flex flex-col border-b border-primary-400 mb-1 leading-loose">
                <h2 className="font-700" data-testid="tooltip-title">
                    {title}
                </h2>
                {subtitle && <span data-testid="tooltip-subtitle">{subtitle}</span>}
            </div>
            <div className="text-left pt-2" data-testid="tooltip-body">
                {body}
            </div>
            {!!footer && (
                <footer
                    className="pt-2 font-700 text-left leading-loose"
                    data-testid="tooltip-footer"
                >
                    {footer}
                </footer>
            )}
        </TooltipOverlay>
    );
};

DetailedTooltipOverlay.propTypes = {
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired,
    subtitle: PropTypes.string,
    footer: PropTypes.node,
    className: PropTypes.string,
};

DetailedTooltipOverlay.defaultProps = {
    subtitle: '',
    footer: '',
    className: '',
};

export default DetailedTooltipOverlay;
