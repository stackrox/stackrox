import React from 'react';
import PropTypes from 'prop-types';

const TooltipCardSection = ({ header, children }) => {
    const showChildren = Array.isArray(children) ? children.length > 0 : !!children;
    return (
        <div
            className="bg-tertiary-100 border border-tertiary-400 rounded text-xs"
            data-testid="tooltip-card"
        >
            <header className="border-b border-tertiary-400 font-700 p-2 capitalize">
                {header}
            </header>
            {showChildren && <div className="p-2">{children}</div>}
        </div>
    );
};

TooltipCardSection.propTypes = {
    header: PropTypes.oneOfType([PropTypes.element, PropTypes.string]).isRequired,
    children: PropTypes.node.isRequired,
};

export default TooltipCardSection;
