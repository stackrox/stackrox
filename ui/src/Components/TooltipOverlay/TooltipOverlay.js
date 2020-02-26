import React from 'react';
import PropTypes from 'prop-types';

const TooltipOverlay = ({ className, children }) => (
    <div className={`rox-tooltip-overlay ${className}`}>{children}</div>
);

TooltipOverlay.propTypes = {
    className: PropTypes.string,
    children: PropTypes.node.isRequired
};

TooltipOverlay.defaultProps = {
    className: ''
};

export default TooltipOverlay;
