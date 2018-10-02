import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';

const PanelButton = ({ icon, text, onClick, className, disabled, tooltip }) => {
    const button = (
        <button className={className} onClick={onClick} disabled={disabled}>
            {icon && <span className="flex items-center">{icon}</span>}
            {text && <span className="mx-2">{text}</span>}
        </button>
    );
    if (!tooltip) return button;

    return (
        <Tooltip placement="top" overlay={<div>{tooltip}</div>}>
            <div>{button}</div>
        </Tooltip>
    );
};

PanelButton.propTypes = {
    icon: PropTypes.node,
    text: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    className: PropTypes.string,
    disabled: PropTypes.bool,
    tooltip: PropTypes.string
};

PanelButton.defaultProps = {
    icon: null,
    text: '',
    className: '',
    disabled: false,
    tooltip: ''
};

export default PanelButton;
