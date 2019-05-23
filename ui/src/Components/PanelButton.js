import React from 'react';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';

const PanelButton = ({ icon, text, onClick, className, disabled, tooltip }) => {
    const tooltipText = tooltip || text;
    const tooltipClassName = !tooltip ? 'sm:visible md:invisible' : '';
    return (
        <Tooltip
            placement="top"
            mouseLeaveDelay={0}
            overlay={<div>{tooltipText}</div>}
            overlayClassName={tooltipClassName}
        >
            <div>
                <button type="button" className={className} onClick={onClick} disabled={disabled}>
                    {icon && <span className="flex items-center">{icon}</span>}
                    {text && <span className="mx-2 sm:hidden md:flex">{text}</span>}
                </button>
            </div>
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
