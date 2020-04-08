import React from 'react';
import PropTypes from 'prop-types';

import TooltipOverlay from 'Components/TooltipOverlay';

const DetailedTooltipOverlay = ({ className, title, subtitle, body, footer }) => {
    if (!title || !body) return null;

    return (
        <TooltipOverlay className={`min-w-32 ${className}`}>
            <div className="text-left flex flex-col border-b border-primary-400 mb-1 leading-loose">
                <h2 className="font-700">{title}</h2>
                {subtitle && <span>{subtitle}</span>}
            </div>
            <div className="text-left pt-2">{body}</div>
            {!!footer && (
                <footer className="pt-2 font-700 text-left leading-loose">{footer}</footer>
            )}
        </TooltipOverlay>
    );
};

DetailedTooltipOverlay.propTypes = {
    title: PropTypes.string.isRequired,
    body: PropTypes.node.isRequired,
    subtitle: PropTypes.string,
    footer: PropTypes.node,
    className: PropTypes.string
};

DetailedTooltipOverlay.defaultProps = {
    subtitle: '',
    footer: '',
    className: ''
};

export default DetailedTooltipOverlay;
