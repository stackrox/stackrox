import React from 'react';
import PropTypes from 'prop-types';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

const PanelButton = ({
    children,
    icon,
    text,
    onClick,
    className,
    dataTestId,
    disabled,
    tooltip,
    alwaysVisibleText
}) => {
    const tooltipContent = tooltip || text;
    const tooltipClassName = !tooltip ? 'visible xl:invisible' : '';
    return (
        <Tooltip
            content={<TooltipOverlay>{tooltipContent}</TooltipOverlay>}
            className={tooltipClassName}
        >
            <button
                type="button"
                className={className}
                onClick={onClick}
                disabled={disabled}
                data-testid={dataTestId}
                data-test-id={`${text.toLowerCase()}-button`}
            >
                {icon && <span className="flex items-center">{icon}</span>}
                {children && (
                    <span
                        className={`mx-2 items-center ${
                            alwaysVisibleText ? 'flex' : 'hidden xl:flex'
                        }`}
                    >
                        {children}
                    </span>
                )}
            </button>
        </Tooltip>
    );
};

PanelButton.propTypes = {
    children: PropTypes.node,
    icon: PropTypes.node,
    text: PropTypes.string,
    onClick: PropTypes.func.isRequired,
    className: PropTypes.string,
    dataTestId: PropTypes.string,
    disabled: PropTypes.bool,
    tooltip: PropTypes.node,
    alwaysVisibleText: PropTypes.bool
};

PanelButton.defaultProps = {
    children: null,
    icon: null,
    text: '',
    className: '',
    dataTestId: '',
    disabled: false,
    tooltip: '',
    alwaysVisibleText: false
};

export default PanelButton;
