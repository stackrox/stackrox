import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';

const RowActionButton = ({
    text,
    tooltipPlacement,
    icon,
    border,
    className,
    overlayClassName,
    onClick
}) => (
    <Tooltip
        placement={tooltipPlacement}
        overlay={text}
        overlayClassName={overlayClassName}
        mouseLeaveDelay={0}
    >
        <button type="button" className={`p-1 px-4 ${className} ${border}`} onClick={onClick}>
            {icon}
        </button>
    </Tooltip>
);

RowActionButton.propTypes = {
    text: PropTypes.string.isRequired,
    tooltipPlacement: PropTypes.string,
    icon: PropTypes.element.isRequired,
    border: PropTypes.string,
    className: PropTypes.string,
    overlayClassName: PropTypes.string,
    onClick: PropTypes.func.isRequired
};

RowActionButton.defaultProps = {
    className: 'hover:bg-primary-200 text-primary-600 hover:text-primary-700',
    border: '',
    tooltipPlacement: 'top',
    overlayClassName: ''
};

export default RowActionButton;
